use std::{env, path::PathBuf, process};

use o101::messagesorter;

fn main() {
    if let Err(error) = run() {
        eprintln!("failed to process capture: {error}");
        process::exit(1);
    }
}

fn run() -> o101::Result<()> {
    let mut args = env::args().skip(1);
    let mut output_dir = None;
    let mut input = None;

    while let Some(argument) = args.next() {
        match argument.as_str() {
            "--out" => {
                output_dir = Some(PathBuf::from(
                    args.next().unwrap_or_else(|| usage("--out needs a value")),
                ));
            }
            "-h" | "--help" => usage(""),
            argument if argument.starts_with("--out=") => {
                output_dir = Some(PathBuf::from(&argument[6..]));
            }
            argument if argument.starts_with('-') => usage(&format!("unknown option: {argument}")),
            argument => {
                if input.replace(PathBuf::from(argument)).is_some() {
                    usage("only one capture file may be provided");
                }
            }
        }
    }

    let input = input.unwrap_or_else(|| usage("capture file is required"));
    let (output_path, result) = messagesorter::process_file(&input, output_dir.as_deref())?;
    println!(
        "wrote {} messages for service {} ({}) to {}",
        result.messages.len(),
        result.service_name,
        result.service_id,
        output_path.display()
    );
    Ok(())
}

fn usage(error: &str) -> ! {
    if !error.is_empty() {
        eprintln!("{error}\n");
    }
    eprintln!("usage: messagesorter [--out <directory>] <capture-file>");
    process::exit(if error.is_empty() { 0 } else { 2 });
}
