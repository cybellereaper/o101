use std::{
    env,
    io::{Read, Write},
    net::{TcpListener, TcpStream},
    path::PathBuf,
    process,
    sync::{
        Arc,
        atomic::{AtomicBool, Ordering},
    },
    thread,
    time::Duration,
};

use o101::wizserver::Realm;

fn main() {
    if let Err(error) = run() {
        eprintln!("{error}");
        process::exit(1);
    }
}

fn run() -> o101::Result<()> {
    let options = Options::parse();
    let game_dir = options.game_dir.canonicalize().map_err(|error| {
        o101::Error::message(format!("game directory validation failed: {error}"))
    })?;

    println!(
        "Starting WizTurtle server with data directory {}",
        game_dir.display()
    );
    let realm = Realm::new(
        options.max_players,
        options.zone_count,
        options.zone_capacity,
    );
    println!(
        "Realm initialised with {} zones for up to {} players",
        realm.zones().len(),
        realm.max_players()
    );

    let running = Arc::new(AtomicBool::new(true));
    let signal_running = Arc::clone(&running);
    ctrlc::set_handler(move || signal_running.store(false, Ordering::Relaxed)).map_err(
        |error| o101::Error::message(format!("failed to install signal handler: {error}")),
    )?;

    let login = start_listener(&options.login_addr, "login", Arc::clone(&running))?;
    let game = match start_listener(&options.game_addr, "game", Arc::clone(&running)) {
        Ok(handle) => handle,
        Err(error) => {
            running.store(false, Ordering::Relaxed);
            let _ = login.join();
            return Err(error);
        }
    };

    while running.load(Ordering::Relaxed) {
        thread::sleep(Duration::from_millis(100));
    }

    println!("Shutting down... waiting for active connections to drain");
    let _ = login.join();
    let _ = game.join();
    println!("Server stopped cleanly");
    Ok(())
}

fn start_listener(
    address: &str,
    name: &'static str,
    running: Arc<AtomicBool>,
) -> o101::Result<thread::JoinHandle<()>> {
    let listener = TcpListener::bind(address)
        .map_err(|error| o101::Error::message(format!("failed to listen on {address}: {error}")))?;
    listener.set_nonblocking(true)?;

    Ok(thread::spawn(move || {
        let mut connections = Vec::new();
        while running.load(Ordering::Relaxed) {
            match listener.accept() {
                Ok((stream, _)) => {
                    let connection_running = Arc::clone(&running);
                    connections.push(thread::spawn(move || {
                        handle_connection(stream, name, &connection_running)
                    }));
                }
                Err(error) if error.kind() == std::io::ErrorKind::WouldBlock => {
                    thread::sleep(Duration::from_millis(50));
                }
                Err(error) => {
                    eprintln!("stopping {name} listener: {error}");
                    break;
                }
            }
        }

        for connection in connections {
            let _ = connection.join();
        }
    }))
}

fn handle_connection(mut stream: TcpStream, name: &str, running: &AtomicBool) {
    let timeout = Some(Duration::from_secs(5));
    if let Err(error) = stream.set_read_timeout(timeout) {
        eprintln!("failed to set read timeout for {name} connection: {error}");
        return;
    }
    if let Err(error) = stream.set_write_timeout(timeout) {
        eprintln!("failed to set write timeout for {name} connection: {error}");
        return;
    }

    let greeting = format!("Welcome to the WizTurtle {name} service!\n");
    if let Err(error) = stream.write_all(greeting.as_bytes()) {
        eprintln!("failed to write greeting to {name} client: {error}");
        return;
    }

    let mut buffer = [0_u8; 256];
    let read = match stream.read(&mut buffer) {
        Ok(0) => return,
        Ok(read) => read,
        Err(error)
            if matches!(
                error.kind(),
                std::io::ErrorKind::TimedOut | std::io::ErrorKind::WouldBlock
            ) =>
        {
            return;
        }
        Err(error) => {
            eprintln!("read error on {name} connection: {error}");
            return;
        }
    };

    let input = String::from_utf8_lossy(&buffer[..read]);
    let response = format!("You said: {}", input.trim());
    if let Err(error) = stream.write_all(response.as_bytes()) {
        eprintln!("failed to send response to {name} client: {error}");
        return;
    }

    if running.load(Ordering::Relaxed) {
        thread::sleep(Duration::from_millis(250));
    }
}

struct Options {
    game_dir: PathBuf,
    login_addr: String,
    game_addr: String,
    max_players: u32,
    zone_count: u32,
    zone_capacity: u32,
}

impl Options {
    fn parse() -> Self {
        let mut args = env::args().skip(1);
        let mut game_dir = None;
        let mut login_addr = "127.0.0.1:12500".to_owned();
        let mut game_addr = "127.0.0.1:12501".to_owned();
        let mut max_players = 100;
        let mut zone_count = 50;
        let mut zone_capacity = 10;

        while let Some(argument) = args.next() {
            match argument.as_str() {
                "--game-dir" => game_dir = Some(PathBuf::from(next_value(&mut args, "--game-dir"))),
                "--login-addr" => login_addr = next_value(&mut args, "--login-addr"),
                "--game-addr" => game_addr = next_value(&mut args, "--game-addr"),
                "--max-players" => {
                    max_players =
                        parse_u32(&next_value(&mut args, "--max-players"), "--max-players")
                }
                "--zones" => zone_count = parse_u32(&next_value(&mut args, "--zones"), "--zones"),
                "--zone-capacity" => {
                    zone_capacity =
                        parse_u32(&next_value(&mut args, "--zone-capacity"), "--zone-capacity")
                }
                "-h" | "--help" => usage(""),
                value if value.starts_with("--game-dir=") => {
                    game_dir = Some(PathBuf::from(&value[11..]))
                }
                value if value.starts_with("--login-addr=") => login_addr = value[13..].to_owned(),
                value if value.starts_with("--game-addr=") => game_addr = value[12..].to_owned(),
                value if value.starts_with("--max-players=") => {
                    max_players = parse_u32(&value[14..], "--max-players")
                }
                value if value.starts_with("--zones=") => {
                    zone_count = parse_u32(&value[8..], "--zones")
                }
                value if value.starts_with("--zone-capacity=") => {
                    zone_capacity = parse_u32(&value[16..], "--zone-capacity")
                }
                value => usage(&format!("unknown argument: {value}")),
            }
        }

        Self {
            game_dir: game_dir.unwrap_or_else(|| usage("--game-dir must be provided")),
            login_addr,
            game_addr,
            max_players,
            zone_count,
            zone_capacity,
        }
    }
}

fn next_value(args: &mut impl Iterator<Item = String>, flag: &str) -> String {
    args.next()
        .unwrap_or_else(|| usage(&format!("{flag} needs a value")))
}

fn parse_u32(value: &str, flag: &str) -> u32 {
    value
        .parse()
        .unwrap_or_else(|_| usage(&format!("{flag} must be a non-negative 32-bit integer")))
}

fn usage(error: &str) -> ! {
    if !error.is_empty() {
        eprintln!("{error}\n");
    }
    eprintln!(
        "usage: wizserver --game-dir <dir> [--login-addr <addr>] [--game-addr <addr>] [--max-players <n>] [--zones <n>] [--zone-capacity <n>]"
    );
    process::exit(if error.is_empty() { 0 } else { 2 });
}
