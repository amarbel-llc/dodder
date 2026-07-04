mod add;
mod alfa;
mod init;
mod konfig;
mod prelude;
mod show;

use crate::alfa::compression::Compression;
use crate::alfa::hash::digest::Digest;
use crate::konfig::Konfig;
use clap::{Parser, Subcommand};
use std::error::Error;
use std::path::PathBuf;

use self::alfa::encryption::Type;
use self::konfig::Angeboren;

#[derive(Parser, Debug)]
#[clap(name = "abscess", version)]
struct App {
    #[clap(subcommand)]
    command: Commands,
}

#[derive(Default, Parser, Debug)]
struct CommandInit {
    #[arg(short, long, default_value_t = Compression::None)]
    compression: Compression,

    #[arg(short, long, default_value_t = Type::None)]
    encryption: Type,

    dir: Option<PathBuf>,
}

#[derive(Parser, Debug)]
struct CommandAdd {
    paths: Vec<PathBuf>,
}

#[derive(Parser, Debug)]
struct CommandShow {
    digests: Vec<Digest>,
}

#[derive(Subcommand, Debug)]
enum Commands {
    Init(CommandInit),
    Add(CommandAdd),
    Move(CommandAdd),
    Show(CommandShow),
}

type Result<T> = std::result::Result<T, Box<dyn Error>>;

fn main() -> Result<()> {
    match App::parse().command {
        Commands::Init(cmd) => {
            let mut angeboren = Angeboren::default();
            angeboren.encryption = cmd.encryption;
            angeboren.compression = cmd.compression;
            init::run(cmd.dir, angeboren)
        }
        Commands::Add(cmd) => add::run(cmd.paths, add::Mode::Add, &Konfig::new()?),
        Commands::Move(cmd) => add::run(cmd.paths, add::Mode::Delete, &Konfig::new()?),
        Commands::Show(cmd) => show::run(&cmd.digests, Konfig::new()?),
    }
}
