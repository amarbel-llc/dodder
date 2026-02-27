# object_finalizer

Object finalization with digest calculation, signature, and lockfile generation.

## Purpose

Finalizes SKU objects by computing digests, signing, verifying, and writing lockfiles.

## Key Types

- `Finalizer`: Handles object finalization with optional signing and verification

## Features

- Calculate object digests using repo public key
- Sign objects with private key and verify signatures
- Generate lockfiles for types and tags (prevents version conflicts)
- Supports finalize-only, finalize-and-sign, and finalize-and-verify operations
- Builder pattern for construction with customizable verification options
