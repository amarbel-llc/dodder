use age::IdentityFile;
use age::IdentityFileEntry;
use anyhow::Error;
use std::io::Read;

use age::Encryptor;
use age::stream::StreamWriter;

use super::wrap_io::{WrapIO, WriteFinish};

type Result<T> = std::result::Result<T, Error>;

type BoxedRecipient = Box<dyn age::Recipient + Send>;
type BoxedIdentity = Box<dyn age::Identity>;

pub struct Age {
    identities: Vec<BoxedIdentity>,
    recipients: Vec<BoxedRecipient>,
}

impl Age {
    pub fn with_identity_file(path: String) -> Result<Self> {
        let identity_file = IdentityFile::from_file(path)?;
        let pairs = identity_file.into_identities().into_iter().map(|e| {
            let id = match e {
                IdentityFileEntry::Native(e) => e,
            };

            let recip = id.to_public();

            (
                Box::new(id) as BoxedIdentity,
                Box::new(recip) as BoxedRecipient,
            )
        });

        let (identities, recipients) = pairs.unzip();

        Ok(Age {
            identities,
            recipients,
        })
    }
}

impl WriteFinish for StreamWriter<Box<dyn WriteFinish>> {
    fn finish(self: Box<Self>) -> Result<()> {
        StreamWriter::finish(*self)?;
        Ok(())
    }
}

impl WrapIO for Age {
    fn wrap_input(&self, reader: Box<dyn Read>) -> Result<Box<dyn Read>> {
        Ok(reader)
    }

    fn wrap_output(&self, writer: Box<dyn WriteFinish>) -> Result<Box<dyn WriteFinish>> {
        match Encryptor::with_recipients_ref(&self.recipients) {
            Some(e) => match e.wrap_output(writer) {
                Ok(e) => Ok(Box::new(e)),
                Err(ouch) => Err(ouch.into()),
            },
            None => anyhow::bail!("no recipients"),
        }
    }
}
