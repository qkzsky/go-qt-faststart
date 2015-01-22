package qtfaststart

// An Atom is a discrete component of a Quicktime movie.
type Atom struct {
	Type       string // The type of atom
	Offset     uint64 // The number of bytes between the start of the file and the atom
	Size       uint64 // The length of the atom
	HeaderSize uint8  // The length of the atom's header
}

// End returns the offset of the end of the atom.
func (atom Atom) End() uint64 {
	return atom.Offset + atom.Size
}

// HeaderEnd returns the offset of the end of the atom's header.
func (atom Atom) HeaderEnd() uint64 {
	return atom.Offset + uint64(atom.HeaderSize)
}

// IsZero returns true if the atom has the zero value for its type.
func (atom Atom) IsZero() bool {
	return atom == Atom{}
}
