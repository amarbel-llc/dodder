use std::path::PathBuf;

use clap::Parser;

use crate::alfa::hash::digest::Digest;

#[derive(Parser, Debug)]
struct Init {
    dir: Option<PathBuf>,
}

#[derive(Parser, Debug)]
struct Add {
    paths: Vec<PathBuf>,
}

#[derive(Parser, Debug)]
struct Show {
    shas: Vec<Digest>,
}

enum AddMode {
    Add,
    Delete,
}
