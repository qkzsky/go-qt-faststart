package qtfaststart

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// A File represents a Quicktime movie file.
type File struct {
	*bytes.Reader
	atoms []Atom
	bytes []byte
	ftyp  Atom
	mdat  Atom
	moov  Atom
}

// New creates and initializes a File by parsing the contents of a byte slice.
func New(b []byte) (*File, error) {
	f := &File{
		bytes:  b,
		Reader: bytes.NewReader(b),
	}
	if err := f.parse(); err != nil {
		return nil, err
	}
	if err := f.validate(); err != nil {
		return nil, err
	}
	return f, nil
}

// Read creates and initializes a File using data read from an io.Reader.
func Read(reader io.Reader) (*File, error) {
	buffer := bytes.NewBuffer(nil)
	if _, err := io.Copy(buffer, reader); err != nil {
		return nil, err
	}
	return New(buffer.Bytes())
}

// FastStartEnabled returns true if the "moov" atom is already at the beginning
// of the file.
func (f *File) FastStartEnabled() bool {
	return f.moov.Offset < f.mdat.Offset
}

// Convert rearranges the file such that the "moov" atom is as close as possible
// to the beginning of the file.
func (f *File) Convert(clean bool) error {
	var shiftBy uint64
	if clean {
		for _, atom := range f.atoms {
			if atom.Type == "free" && atom.Offset < f.mdat.Offset {
				shiftBy += atom.Size
			} else if atom.Type == "\x00\x00\x00\x00" && atom.Offset < f.mdat.Offset {
				shiftBy += atom.Size
			}
		}
	}

	if f.moov.Offset > f.mdat.Offset {
		shiftBy += f.moov.Size
	}

	patchedMoov, err := f.patchChunkOffsetAtoms(f.moov, shiftBy)
	if err != nil {
		return err
	}

	out := bytes.NewBuffer(nil)

	// Copy the ftyp atom first
	err = f.copyAtom(f.ftyp, out)
	if err != nil {
		return err
	}

	// Then the patched moov atom
	if _, err = out.Write(patchedMoov); err != nil {
		return err
	}

	// And then everything else
	for _, atom := range f.atoms {
		if atom.Type == "ftyp" || atom.Type == "moov" {
			continue
		}
		if clean && (atom.Type == "free" || atom.Type == "\x00\x00\x00\x00") {
			continue
		}
		err := f.copyAtom(atom, out)
		if err != nil {
			return err
		}
	}

	f.bytes = out.Bytes()
	f.Reader = bytes.NewReader(f.bytes)
	return nil
}

func (f *File) parse() error {
	offset := uint64(0)
	length := uint64(len(f.bytes))
	for offset < length {
		if atom, err := f.readAtom(offset); err == nil {
			f.atoms = append(f.atoms, atom)
			offset += atom.Size
			switch atom.Type {
			case "ftyp":
				f.ftyp = atom
			case "mdat":
				f.mdat = atom
			case "moov":
				f.moov = atom
			case "free", "junk", "pict", "pnot", "skip", "uuid", "wide":
				// Do nothing
			default:
				return fmt.Errorf("%q is not a valid top-level atom")
			}
		} else {
			return err
		}
	}
	return nil
}

func (f *File) validate() error {
	if f.ftyp.IsZero() {
		return errors.New("Invalid file: ftyp atom not found")
	}
	if f.mdat.IsZero() {
		return errors.New("Invalid file: mdat atom not found")
	}
	if f.moov.IsZero() {
		return errors.New("Invalid file: moov atom not found")
	}
	if yes, err := f.containsCompressedAtoms(f.moov); err != nil {
		return err
	} else if yes {
		return errors.New("Compressed moov atoms are not supported")
	}
	return nil
}

func (f *File) containsCompressedAtoms(parent Atom) (bool, error) {
	offset := parent.Offset
	for offset < parent.Size {
		if child, err := f.readAtom(offset); err == nil {
			if child.Type == "cmov" {
				return true, nil
			}
			offset += child.Size
		} else {
			return false, err
		}
	}
	return false, nil
}

func (f *File) readAtom(offset uint64) (Atom, error) {
	atom := Atom{Offset: offset, HeaderSize: 8}

	// The first 4 bytes contain the size of the atom
	atom.Size = uint64(binary.BigEndian.Uint32(f.bytes[offset : offset+4]))

	// The next 4 bytes contain the type of the atom
	atom.Type = string(f.bytes[offset+4 : offset+8])

	// If the size is 1, look at the next 8 bytes for the real size
	if atom.Size == 1 {
		atom.Size = binary.BigEndian.Uint64(f.bytes[offset+8 : offset+16])
		atom.HeaderSize += 8
	}

	// Make sure the atom is at least as large as its header
	if atom.Size < uint64(atom.HeaderSize) {
		return atom, fmt.Errorf("Invalid file format: Atom %s at offset %i reported a size of only %i bytes", atom.Type, atom.Offset, atom.Size)
	}

	return atom, nil
}

func (f *File) copyAtom(atom Atom, dest io.Writer) error {
	start, end := atom.Offset, atom.End()
	_, err := dest.Write(f.bytes[start:end])
	return err
}

func (f *File) patchChunkOffsetAtoms(parent Atom, shiftBy uint64) ([]byte, error) {
	atoms, err := f.findChunkOffsetAtoms(parent)
	if err != nil {
		return nil, err
	}

	// Make a copy of the bytes for this atom
	parentBytes := make([]byte, parent.Size)
	for i := uint64(0); i < parent.Size; i++ {
		parentBytes[i] = f.bytes[parent.Offset+i]
	}

	for _, atom := range atoms {
		// Ignore the first 4 bytes after the header
		// The next 4 bytes tells us how many entries need to be patched
		offset := atom.HeaderEnd() - parent.Offset + 8
		numEntries := uint64(binary.BigEndian.Uint32(parentBytes[offset-4 : offset]))

		var i uint64
		if atom.Type == "stco" {
			for i = 0; i < numEntries; i++ {
				start, end := offset+4*i, offset+4*(i+1)
				entryBytes := parentBytes[start:end]
				entry := binary.BigEndian.Uint32(entryBytes)
				entry += uint32(shiftBy)
				binary.BigEndian.PutUint32(entryBytes, entry)
			}
		} else if atom.Type == "co64" {
			for i = 0; i < numEntries; i++ {
				start, end := offset+8*i, offset+8*(i+1)
				entryBytes := parentBytes[start:end]
				entry := binary.BigEndian.Uint64(entryBytes)
				entry += shiftBy
				binary.BigEndian.PutUint64(entryBytes, entry)
			}
		}
	}

	return parentBytes, nil
}

func (f *File) findChunkOffsetAtoms(parent Atom) ([]Atom, error) {
	var err error
	atoms := []Atom{}
	offset := parent.HeaderEnd()
	for offset < parent.End() {
		child, err := f.readAtom(offset)
		if err == nil {
			switch child.Type {
			case "stco", "co64":
				atoms = append(atoms, child)
			case "trak", "mdia", "minf", "stbl": // These can have chunk offset atoms as children
				var children []Atom
				children, err = f.findChunkOffsetAtoms(child)
				if err == nil {
					atoms = append(atoms, children...)
				}
			}
			offset += child.Size
		}
	}
	return atoms, err
}
