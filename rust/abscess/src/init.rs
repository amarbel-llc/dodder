use std::env::current_dir;
use std::error::Error;
use std::fs::create_dir;
use std::io;
use std::path::PathBuf;

use crate::konfig::{Angeboren, Konfig};

type Result<T> = std::result::Result<T, Box<dyn Error>>;

pub fn run(dir: Option<PathBuf>, angeboren: Angeboren) -> Result<()> {
    let result = create_dir(match dir {
        Some(dir) => dir,
        None => current_dir()?,
    });

    if let Err(ref ouch) = result {
        match ouch.kind() {
            io::ErrorKind::AlreadyExists => (),
            _ => result?,
        }
    }

    let k = Konfig::from_angeboren(angeboren);
    k.angeboren.write_to_default_location()?;

    Ok(())
}
