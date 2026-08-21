use std::{
    env,
    path::PathBuf,
    process,
    sync::{
        Arc,
        atomic::{AtomicBool, Ordering},
    },
    time::Instant,
};

use o101::{
    patcher::{Config, Logger, PatchOutcome, Patcher},
    state::Store,
};

fn main() {
    if let Err(error) = run() {
        eprintln!("patch failed: {error}");
        process::exit(1);
    }
}

fn run() -> o101::Result<()> {
    let options = Options::parse();
    let state_file = options
        .state_file
        .unwrap_or_else(|| options.install_dir.join(".wizturtle").join("state.json"));

    let logger: Option<Arc<dyn Logger>> = if options.quiet {
        None
    } else {
        Some(Arc::new(|message: &str| println!("wizturtle: {message}")))
    };

    let patcher = Patcher::new(Config {
        patch_info_url: options.patch_info,
        install_dir: options.install_dir,
        state_store: Arc::new(Store::new(state_file)),
        http_client: None,
        concurrency: options.concurrency,
        logger,
    })?;

    let cancelled = Arc::new(AtomicBool::new(false));
    let signal_cancelled = Arc::clone(&cancelled);
    ctrlc::set_handler(move || signal_cancelled.store(true, Ordering::Relaxed)).map_err(
        |error| o101::Error::message(format!("failed to install signal handler: {error}")),
    )?;

    let started = Instant::now();
    match patcher.run(cancelled.as_ref())? {
        PatchOutcome::UpToDate => {
            if !options.quiet {
                println!("Installation is already up to date.");
            }
        }
        PatchOutcome::Updated => {
            if !options.quiet {
                println!("Patch completed in {:.3}s", started.elapsed().as_secs_f64());
            }
        }
    }
    Ok(())
}

struct Options {
    patch_info: String,
    install_dir: PathBuf,
    state_file: Option<PathBuf>,
    concurrency: usize,
    quiet: bool,
}

impl Options {
    fn parse() -> Self {
        let mut args = env::args().skip(1);
        let mut patch_info = None;
        let mut install_dir = PathBuf::from(".");
        let mut state_file = None;
        let mut concurrency = 0;
        let mut quiet = false;

        while let Some(argument) = args.next() {
            match argument.as_str() {
                "--patch-info" => patch_info = Some(next_value(&mut args, "--patch-info")),
                "--install-dir" => {
                    install_dir = PathBuf::from(next_value(&mut args, "--install-dir"))
                }
                "--state-file" => {
                    state_file = Some(PathBuf::from(next_value(&mut args, "--state-file")))
                }
                "--concurrency" => {
                    concurrency =
                        parse_usize(&next_value(&mut args, "--concurrency"), "--concurrency")
                }
                "--quiet" => quiet = true,
                "-h" | "--help" => usage(""),
                value if value.starts_with("--patch-info=") => {
                    patch_info = Some(value[13..].to_owned())
                }
                value if value.starts_with("--install-dir=") => {
                    install_dir = PathBuf::from(&value[14..])
                }
                value if value.starts_with("--state-file=") => {
                    state_file = Some(PathBuf::from(&value[13..]))
                }
                value if value.starts_with("--concurrency=") => {
                    concurrency = parse_usize(&value[14..], "--concurrency")
                }
                value => usage(&format!("unknown argument: {value}")),
            }
        }

        Self {
            patch_info: patch_info.unwrap_or_else(|| usage("--patch-info is required")),
            install_dir,
            state_file,
            concurrency,
            quiet,
        }
    }
}

fn next_value(args: &mut impl Iterator<Item = String>, flag: &str) -> String {
    args.next()
        .unwrap_or_else(|| usage(&format!("{flag} needs a value")))
}

fn parse_usize(value: &str, flag: &str) -> usize {
    value
        .parse()
        .unwrap_or_else(|_| usage(&format!("{flag} must be a non-negative integer")))
}

fn usage(error: &str) -> ! {
    if !error.is_empty() {
        eprintln!("{error}\n");
    }
    eprintln!(
        "usage: wizturtle --patch-info <url> [--install-dir <dir>] [--state-file <path>] [--concurrency <n>] [--quiet]"
    );
    process::exit(if error.is_empty() { 0 } else { 2 });
}
