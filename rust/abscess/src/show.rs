use crate::alfa::hash::digest::Digest;
use crate::alfa::wrap_io::WrapIO;
use crate::konfig::Konfig;
use std::error::Error;
use std::fs::{File, OpenOptions};
use std::io::{BufReader, Write, copy, stdout};

type Result<T> = std::result::Result<T, Box<dyn Error>>;

fn file_for_digest(dig: &Digest) -> Result<File> {
    let file = OpenOptions::new().read(true).open(dig.path())?;

    Ok(file)
}

fn write_file_to<T: Write>(file: File, out: &mut T, konfig: Konfig) -> Result<()> {
    let mut reader = konfig
        .angeboren
        .wrap_input(Box::new(BufReader::new(file)))?;
    copy(&mut reader, out)?;

    Ok(())
}

pub fn run(digs: &Vec<Digest>, konfig: Konfig) -> Result<()> {
    let mut out = stdout();
    for dig in digs.iter() {
        if dig.is_null() {
            continue;
        }

        let file = file_for_digest(&dig)?;
        write_file_to(file, &mut out, konfig.to_owned())?;
    }

    Ok(())
}
