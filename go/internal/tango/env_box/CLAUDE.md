# env_box

Provides formatted output and printing facilities for SKU objects (Transacted and CheckedOut).

## Key Types

- `Env`: Interface wrapping env_repo with box formatters and printers
- `env`: Implementation combining repo environment, config, store, and abbreviation index

## Features

- StringFormatWriterSkuBoxTransacted: Creates formatted box output for transacted SKUs with color/truncation options
- StringFormatWriterSkuBoxCheckedOut: Creates formatted box output for checked-out SKUs
- PrinterTransacted: Returns iterator for printing transacted SKUs to UI
- PrinterCheckedOut: Returns iterator for printing checked-out SKUs to UI
- GetUIStorePrinters: Provides printer bundle for new/updated/unchanged transacted and checked-out objects
