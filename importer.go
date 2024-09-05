package gofpdi

import (
	"fmt"
	"io"
)

// The Importer class to be used by a pdf generation library
type Importer struct {
	sourceFile    string
	readers       map[string]*PdfReader
	writers       map[string]*PdfWriter
	tplMap        map[int]*TplInfo
	tplN          int
	writer        *PdfWriter
	importedPages map[string]int
}

type TplInfo struct {
	SourceFile string
	Writer     *PdfWriter
	TemplateId int
}

func (imp *Importer) GetReader() *PdfReader {
	return imp.GetReaderForFile(imp.sourceFile)
}

func (imp *Importer) GetWriter() *PdfWriter {
	return imp.GetWriterForFile(imp.sourceFile)
}

func (imp *Importer) GetReaderForFile(file string) *PdfReader {
	if _, ok := imp.readers[file]; ok {
		return imp.readers[file]
	}

	return nil
}

func (imp *Importer) GetWriterForFile(file string) *PdfWriter {
	if _, ok := imp.writers[file]; ok {
		return imp.writers[file]
	}

	return nil
}

func NewImporter() *Importer {
	importer := &Importer{}
	importer.init()

	return importer
}

func (imp *Importer) init() {
	imp.readers = make(map[string]*PdfReader, 0)
	imp.writers = make(map[string]*PdfWriter, 0)
	imp.tplMap = make(map[int]*TplInfo, 0)
	imp.writer, _ = NewPdfWriter("")
	imp.importedPages = make(map[string]int, 0)
}

func (imp *Importer) SetSourceFile(f string) {
	imp.sourceFile = f

	// If reader hasn't been instantiated, do that now
	if _, ok := imp.readers[imp.sourceFile]; !ok {
		reader, err := NewPdfReader(imp.sourceFile)
		if err != nil {
			panic(err)
		}
		imp.readers[imp.sourceFile] = reader
	}

	// If writer hasn't been instantiated, do that now
	if _, ok := imp.writers[imp.sourceFile]; !ok {
		writer, err := NewPdfWriter("")
		if err != nil {
			panic(err)
		}

		// Make the next writer start template numbers at imp.tplN
		writer.SetTplIdOffset(imp.tplN)
		imp.writers[imp.sourceFile] = writer
	}
}

func (imp *Importer) SetSourceStream(rs *io.ReadSeeker) {
	imp.sourceFile = fmt.Sprintf("%v", rs)

	if _, ok := imp.readers[imp.sourceFile]; !ok {
		reader, err := NewPdfReaderFromStream(imp.sourceFile, *rs)
		if err != nil {
			panic(err)
		}
		imp.readers[imp.sourceFile] = reader
	}

	// If writer hasn't been instantiated, do that now
	if _, ok := imp.writers[imp.sourceFile]; !ok {
		writer, err := NewPdfWriter("")
		if err != nil {
			panic(err)
		}

		// Make the next writer start template numbers at imp.tplN
		writer.SetTplIdOffset(imp.tplN)
		imp.writers[imp.sourceFile] = writer
	}
}

func (imp *Importer) GetNumPages() int {
	result, err := imp.GetReader().getNumPages()
	if err != nil {
		panic(err)
	}

	return result
}

func (imp *Importer) GetPageSizes() map[int]map[string]map[string]float64 {
	result, err := imp.GetReader().getAllPageBoxes(1.0)
	if err != nil {
		panic(err)
	}

	return result
}

func (imp *Importer) ImportPage(pageno int, box string) int {
	// If page has already been imported, return existing tplN
	pageNameNumber := fmt.Sprintf("%s-%04d", imp.sourceFile, pageno)
	if _, ok := imp.importedPages[pageNameNumber]; ok {
		return imp.importedPages[pageNameNumber]
	}

	res, err := imp.GetWriter().ImportPage(imp.GetReader(), pageno, box)
	if err != nil {
		panic(err)
	}

	// Get current template id
	tplN := imp.tplN

	// Set tpl info
	imp.tplMap[tplN] = &TplInfo{SourceFile: imp.sourceFile, TemplateId: res, Writer: imp.GetWriter()}

	// Increment template id
	imp.tplN++

	// Cache imported page tplN
	imp.importedPages[pageNameNumber] = tplN

	return tplN
}

func (imp *Importer) SetNextObjectID(objId int) {
	imp.GetWriter().SetNextObjectID(objId)
}

// PutFormXobjects Put form xobjects and get back a map of template names (e.g.
// /GOFPDITPL1) and their object ids (int)
func (imp *Importer) PutFormXobjects() map[string]int {
	res := make(map[string]int, 0)
	tplNamesIds, err := imp.GetWriter().PutFormXobjects(imp.GetReader())
	if err != nil {
		panic(err)
	}
	for tplName, pdfObjId := range tplNamesIds {
		res[tplName] = pdfObjId.id
	}
	return res
}

// PutFormXobjectsUnordered Put form xobjects and get back a map of template
// names (e.g. /GOFPDITPL1) and their object ids (sha1 hash)
func (imp *Importer) PutFormXobjectsUnordered() map[string]string {
	imp.GetWriter().SetUseHash(true)
	res := make(map[string]string, 0)
	tplNamesIds, err := imp.GetWriter().PutFormXobjects(imp.GetReader())
	if err != nil {
		panic(err)
	}
	for tplName, pdfObjId := range tplNamesIds {
		res[tplName] = pdfObjId.hash
	}
	return res
}

// GetImportedObjects Get object ids (int) and their contents (string)
func (imp *Importer) GetImportedObjects() map[int]string {
	res := make(map[int]string, 0)
	pdfObjIdBytes := imp.GetWriter().GetImportedObjects()
	for pdfObjId, bytes := range pdfObjIdBytes {
		res[pdfObjId.id] = string(bytes)
	}
	return res
}

// GetImportedObjectsUnordered Get object ids (sha1 hash) and their contents
// ([]byte). The contents may have references to other object hashes which will
// need to be replaced by the pdf generator library. The positions of the hashes
// (sha1 - 40 characters) can be obtained by calling GetImportedObjHashPos().
func (imp *Importer) GetImportedObjectsUnordered() map[string][]byte {
	res := make(map[string][]byte, 0)
	pdfObjIdBytes := imp.GetWriter().GetImportedObjects()
	for pdfObjId, bytes := range pdfObjIdBytes {
		res[pdfObjId.hash] = bytes
	}
	return res
}

// GetImportedObjHashPos Get the positions of the hashes (sha1 - 40 characters)
// within each object, to be replaced with actual objects ids by the pdf
// generator library
func (imp *Importer) GetImportedObjHashPos() map[string]map[int]string {
	res := make(map[string]map[int]string, 0)
	pdfObjIdPosHash := imp.GetWriter().GetImportedObjHashPos()
	for pdfObjId, posHashMap := range pdfObjIdPosHash {
		res[pdfObjId.hash] = posHashMap
	}
	return res
}

// UseTemplate For a given template id (returned from ImportPage), get the
// template name (e.g. /GOFPDITPL1) and the 4 float64 values necessary to draw
// the template a x,y for a given width and height.
func (imp *Importer) UseTemplate(tplid int, _x float64, _y float64, _w float64, _h float64) (string, float64, float64, float64, float64) {
	// Look up template id in importer tpl map
	tplInfo := imp.tplMap[tplid]
	return tplInfo.Writer.UseTemplate(tplInfo.TemplateId, _x, _y, _w, _h)
}
