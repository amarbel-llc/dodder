use crate::alfa::hash::digest::Digest;
use crate::alfa::hash::writer::Writer;
use crate::alfa::wrap_io::WrapIO;
use crate::konfig::Konfig;
use io_tee::TeeWriter;
use rand::{Rng, distributions::Alphanumeric};
use std::error::Error;
use std::fs::{File, create_dir, rename};
use std::fs::{OpenOptions, remove_file};
use std::io::{self, BufReader, BufWriter, ErrorKind, Read, copy, stdin};
use std::path::{Path, PathBuf};
use std::time::SystemTime;

static DIR_TMP: &str = ".tmp";

type Result<T> = std::result::Result<T, Box<dyn Error>>;

pub enum Mode {
    Add,
    Delete,
}

pub fn create_temp_dir_if_necessary() -> Result<()> {
    let result = create_dir(DIR_TMP);

    if let Err(ref ouch) = result {
        match ouch.kind() {
            ErrorKind::AlreadyExists => (),
            _ => result?,
        }
    }

    Ok(())
}

fn create_unique_temp_file() -> Result<(PathBuf, File)> {
    let now = SystemTime::now().duration_since(SystemTime::UNIX_EPOCH)?;
    let s: String = rand::thread_rng()
        .sample_iter(&Alphanumeric)
        .take(7)
        .map(char::from)
        .collect();

    let unique = format!("{}-{}", now.as_micros(), s);

    let path = PathBuf::from(DIR_TMP).join(unique);

    let file = OpenOptions::new()
        .write(true)
        .create_new(true)
        .open(&path)?;

    Ok((path, file))
}

fn copy_file_to_temp_and_generate_digest<T: Read>(
    input: &mut T,
    output: File,
    konfig: &Konfig,
) -> Result<Digest> {
    let mut reader = BufReader::new(input);
    let writer = konfig
        .angeboren
        .wrap_output(Box::new(BufWriter::new(output)))?;
    let mut hash_writer = Writer::new();
    let mut tee_writer = TeeWriter::new(writer, &mut hash_writer);

    copy(&mut reader, &mut tee_writer)?;

    Ok(hash_writer.digest())
}

fn create_directory_if_necessary(dig: &mut Digest) -> Result<()> {
    let path = Path::new(".").join(dig.kopf());

    let result = create_dir(path);

    if let Err(ref ouch) = result {
        match ouch.kind() {
            io::ErrorKind::AlreadyExists => (),
            _ => result?,
        }
    }

    Ok(())
}

fn move_file_to_store(old_path: PathBuf, dig: &mut Digest) -> Result<()> {
    rename(old_path, dig.path())?;

    Ok(())
}

fn run_one<T: Read>(input: &mut T, konfig: &Konfig) -> Result<Digest> {
    let (path, file) = create_unique_temp_file()?;
    let mut dig = copy_file_to_temp_and_generate_digest(input, file, &konfig)?;
    create_directory_if_necessary(&mut dig)?;
    move_file_to_store(path, &mut dig)?;
    Ok(dig)
}

pub fn run(paths: Vec<PathBuf>, add_mode: Mode, konfig: &Konfig) -> Result<()> {
    create_temp_dir_if_necessary()?;

    if paths.len() == 0 {
        let dig = run_one(&mut stdin(), konfig)?;
        println!("{:} (stdin)", dig);
    } else {
        for path in paths.iter() {
            let mut file = OpenOptions::new().read(true).open(&path)?;
            let dig = run_one(&mut file, konfig)?;

            let path_str = path.to_string_lossy();

            match add_mode {
                Mode::Delete => {
                    remove_file(path)?;
                    println!("{:} {:} (deleted)", dig, path_str);
                }
                _ => println!("{:} {:}", dig, path.to_string_lossy()),
            }
        }
    }

    Ok(())
}
