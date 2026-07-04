use pcsc::*;
use std::fs::OpenOptions;
use std::fs::read_to_string;
use std::io::BufWriter;
use std::io::Read;
use std::io::Write;

use anyhow::{Context, Error};
use serde::{Deserialize, Serialize};

use crate::alfa::age::Age;
use crate::alfa::compression::Compression;
use crate::alfa::encryption::Type;
use crate::alfa::wrap_io::NopWrapIO;
use crate::alfa::wrap_io::WrapIO;
use crate::alfa::wrap_io::WriteFinish;

type Result<T> = std::result::Result<T, Error>;

static FILE_KONFIG_ANGEBOREN: &str = "KonfigAngeboren.toml";
static FILE_KONFIG_ENCRYPTION_AGE_KEY: &str = "AgeKey.secret";

#[derive(Default, Serialize, Deserialize, Debug, Clone)]
/// Configuration for an Abscess store.
pub struct Konfig {
    pub angeboren: Angeboren,
    pub erworben: Erworben,
}

impl Konfig {
    pub fn from_angeboren(angeboren: Angeboren) -> Self {
        Konfig {
            angeboren,
            erworben: Erworben::default(),
        }
    }

    pub fn new() -> Result<Self> {
        Ok(Konfig {
            angeboren: Angeboren::read_from_default_location()?,
            erworben: Erworben::default(),
        })
    }
}

#[derive(Default, Serialize, Deserialize, Debug, Clone)]
/// configuration that is "congenital", in that once an Abscess store is formed, this configuration
/// cannot ever be changed. To change it, you must create a new store with the desired
/// configuration, and move objects to the new store.
pub struct Angeboren {
    pub compression: Compression,
    pub encryption: Type,
}

#[derive(Default, Serialize, Deserialize, Debug, Clone)]
/// configuration that is "acquired", in that once an Abscess store is formed, this configuration
/// can changed by providing the appropriate CLI flags.
pub struct Erworben {}

impl Angeboren {
    pub fn read_from_default_location() -> Result<Self> {
        let s = read_to_string(FILE_KONFIG_ANGEBOREN)?;
        Ok(toml::from_str::<Self>(&s)?)
    }

    pub fn write_to_default_location(&self) -> Result<()> {
        let s = toml::to_string_pretty(self)?;
        let file = OpenOptions::new()
            .write(true)
            .create_new(true)
            .open(&FILE_KONFIG_ANGEBOREN)?;

        let mut writer = BufWriter::new(file);

        write!(&mut writer, "{}", s)?;

        let yk = yubikey::YubiKey::open().context("failed to open yubikey")?;
        println!("{:?}", yk);
        // let ctx = Context::establish(Scope::User).expect("failed to establish context");

        // // List connected readers.
        // let readers = ctx.list_readers_owned().expect("failed to list readers");
        // println!("Readers: {:?}", readers);

        // // Try to connect to a card in the first reader.
        // let mut card = ctx
        //     .connect(&readers[0], ShareMode::Shared, Protocols::ANY)
        //     .expect("failed to connect to card");
        // todo!("generate the apporpatie keyfile given the encryption type and fix those typos!!!!");

        Ok(())
    }

    pub fn initialize_encryption_so_that_it_may_be_used_after_initalization_yes_i_know_its_spelled_write(
        &self,
    ) -> Result<Box<dyn WrapIO>> {
        match self.encryption {
            Type::None => Ok(Box::new(NopWrapIO {})),
            Type::Age => Ok(Box::new(Age::with_identity_file(
                FILE_KONFIG_ENCRYPTION_AGE_KEY.to_string(),
            )?)),
        }
    }
}

impl WrapIO for Angeboren {
    fn wrap_input(&self, reader: Box<dyn Read>) -> Result<Box<dyn Read>> {
        self.compression.wrap_input(reader)
        // self.encryption
        //     .wrap_input(self.compression.wrap_input(reader))
    }

    fn wrap_output(&self, writer: Box<dyn WriteFinish>) -> Result<Box<dyn WriteFinish>> {
        self.compression.wrap_output(writer)
        // self.encryption
        //     .wrap_output(self.compression.wrap_output(Box::new(writer)))
    }
}
