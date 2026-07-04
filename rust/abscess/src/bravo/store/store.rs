type Result<T> = std::result::Result<T, Box<dyn Error>>;

mod Init {
    use std::env::current_dir;
    use std::fs::create_dir;

    fn run(dir: String) -> Result<()> {
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

        Ok(())
    }
}

mod Add {
    use std::fs::{File, OpenOptions, create_dir};
    use std::io::{self, BufReader, BufWriter, Read, copy};
    use std::path::{Path, PathBuf};
    use std::time::SystemTime;

    use io_tee::TeeWriter;
    use rand::distributions::Alphanumeric;

    use crate::alfa::hash::digest::Digest;
    use crate::alfa::hash::writer::Writer;

    static DIR_TMP: &str = ".tmp";

    fn create_temp_dir_if_necessary() -> Result<()> {
        let result = create_dir(DIR_TMP);

        if let Err(ref ouch) = result {
            match ouch.kind() {
                io::ErrorKind::AlreadyExists => (),
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

    fn copy_file_to_temp_and_generate_sha<T: Read>(input: &mut T, output: File) -> Result<Digest> {
        let mut reader = BufReader::new(input);
        let writer = BufWriter::new(output);
        let mut hash_writer = Writer::new();
        let mut tee_writer = TeeWriter::new(writer, &mut hash_writer);

        copy(&mut reader, &mut tee_writer)?;

        Ok(hash_writer.digest())
    }

    fn create_directory_if_necessary(sha: &mut Digest) -> Result<()> {
        let path = Path::new(".").join(sha.kopf());

        let result = create_dir(path);

        if let Err(ref ouch) = result {
            match ouch.kind() {
                io::ErrorKind::AlreadyExists => (),
                _ => result?,
            }
        }

        Ok(())
    }

    fn move_file_to_store(old_path: PathBuf, sha: &mut Digest) -> Result<()> {
        fs::rename(old_path, sha.path())?;

        Ok(())
    }

    fn run_one<T: Read>(input: &mut T) -> Result<Digest> {
        let (path, file) = Self::create_unique_temp_file()?;
        let mut sha = Self::copy_file_to_temp_and_generate_sha(input, file)?;
        Self::create_directory_if_necessary(&mut sha)?;
        Self::move_file_to_store(path, &mut sha)?;
        Ok(sha)
    }

    fn run(&self, add_mode: AddMode) -> Result<()> {
        Self::create_temp_dir_if_necessary()?;

        if self.paths.len() == 0 {
            let sha = Self::run_one(&mut stdin())?;
            println!("{:} (stdin)", sha);
        } else {
            for path in self.paths.iter() {
                let mut file = OpenOptions::new().read(true).open(&path)?;
                let sha = Self::run_one(&mut file)?;

                let path_str = path.to_string_lossy();

                match add_mode {
                    AddMode::Delete => {
                        remove_file(path)?;
                        println!("{:} {:} (deleted)", sha, path_str);
                    }
                    _ => println!("{:} {:}", sha, path_str),
                }
            }
        }

        Ok(())
    }
}

mod Show {
    fn file_for_sha(sha: &Digest) -> Result<fs::File> {
        let mut path = PathBuf::new();
        sha.add_to_path(&mut path);

        let file = OpenOptions::new().read(true).open(&path)?;

        Ok(file)
    }

    fn write_file_to<T: Write>(file: fs::File, out: &mut T) -> Result<()> {
        let mut reader = BufReader::new(file);
        copy(&mut reader, out)?;

        Ok(())
    }

    fn run(&self) -> Result<()> {
        let mut out = stdout();
        for sha in self.shas.iter() {
            if sha.is_null() {
                continue;
            }

            let file = Self::file_for_sha(&sha)?;
            Self::write_file_to(file, &mut out)?;
        }

        Ok(())
    }
}
