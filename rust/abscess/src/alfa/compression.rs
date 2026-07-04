use anyhow::Error;
use flate2::Compression as NOT_OUR_COMPRESSION;
use flate2::read::DeflateDecoder;
use flate2::write::DeflateEncoder;
use serde::Deserialize;
use serde::Serialize;
use std::fmt;
use std::fmt::Display;
use std::fmt::Formatter;
use std::io::Read;

use super::wrap_io::WrapIO;
use super::wrap_io::WriteFinish;

type Result<T> = std::result::Result<T, Error>;

#[derive(Serialize, Deserialize, Debug, Clone, clap::ValueEnum)]
pub enum Compression {
    None,
    Gzip,
}

impl Display for Compression {
    fn fmt(&self, f: &mut Formatter) -> fmt::Result {
        match *self {
            Self::None => write!(f, "none"),
            Self::Gzip => write!(f, "gzip"),
        }
    }
}

impl Default for Compression {
    fn default() -> Self {
        Compression::Gzip
    }
}

impl WrapIO for Compression {
    fn wrap_input<'a>(&self, reader: Box<dyn Read + 'a>) -> Result<Box<dyn Read + 'a>> {
        Ok(match self {
            Self::None => Box::new(reader),
            Self::Gzip => Box::new(DeflateDecoder::new(reader)),
        })
    }

    fn wrap_output(&self, writer: Box<dyn WriteFinish>) -> Result<Box<dyn WriteFinish>> {
        Ok(match self {
            Self::None => writer,
            Self::Gzip => Box::new(DeflateEncoder::new(writer, NOT_OUR_COMPRESSION::default())),
        })
    }
}
