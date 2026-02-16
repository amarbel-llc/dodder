package sku_fmt

import (
	"bufio"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/india/env_local"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

type pooledTransacted struct {
	object *sku.Transacted
	repool interfaces.FuncRepool
}

type PrinterComplete struct {
	bufferedWriter *bufio.Writer
	pool           interfaces.PoolPtr[sku.Transacted, *sku.Transacted]
	chObjects      chan pooledTransacted
	chDone         chan struct{}
}

func MakePrinterComplete(envLocal env_local.Env) *PrinterComplete {
	printer := &PrinterComplete{
		chObjects:      make(chan pooledTransacted),
		chDone:         make(chan struct{}),
		bufferedWriter: bufio.NewWriter(envLocal.GetUIFile()),
		pool: pool.Make[sku.Transacted](
			nil,
			nil,
		),
	}

	envLocal.After(printer.Close)

	go func() {
		for pooled := range printer.chObjects {
			object := pooled.object

			ui.TodoP4("handle write errors")
			printer.bufferedWriter.WriteString(object.GetObjectId().String())
			printer.bufferedWriter.WriteByte('\t')

			g := object.GetObjectId().GetGenre()
			printer.bufferedWriter.WriteString(g.String())

			tipe := object.GetType().String()

			if tipe != "" {
				printer.bufferedWriter.WriteString(": ")
				printer.bufferedWriter.WriteString(object.GetType().String())
			}

			description := object.GetMetadataMutable().GetDescription().String()

			if description != "" {
				printer.bufferedWriter.WriteString(" ")
				printer.bufferedWriter.WriteString(
					object.GetMetadataMutable().GetDescription().String(),
				)
			}

			printer.bufferedWriter.WriteString("\n")
			pooled.repool()
		}

		printer.chDone <- struct{}{}
	}()

	return printer
}

func (printer *PrinterComplete) PrintOne(
	src *sku.Transacted,
) (err error) {
	if src.GetObjectId().String() == "/" {
		err = errEmptySku
		return err
	}

	dst, dstRepool := printer.pool.GetWithRepool()
	sku.Resetter.ResetWith(dst, src)

	select {
	case <-printer.chDone:
		dstRepool()
		err = errors.MakeErrStopIteration()

	case printer.chObjects <- pooledTransacted{object: dst, repool: dstRepool}:
	}

	return err
}

func (printer *PrinterComplete) Close(
	context interfaces.ActiveContext,
) (err error) {
	close(printer.chObjects)
	<-printer.chDone

	if err = context.Cause(); err != nil {
		err = nil
		return err
	}

	if err = printer.bufferedWriter.Flush(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
