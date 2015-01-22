// Package qtfaststart implements functions for optimizing Quicktime movie files
// for network streaming.
//
// Quicktime files are composed of atoms, data structures that can contain audio
// or video tracks, subtitles, metadata, and even other atoms.  One atom, called
// "moov", contains general information about the movie, such as the number of
// tracks and where in the file those tracks can be found.  The "moov" atom is
// traditionally placed at the end of the file, but this design decision means
// that playback cannot begin until the entire file has been loaded into memory,
// which translates to significant wait times when playing Quicktime movies on
// the web.  We can solve this problem by rearranging the atoms that make up the
// Quicktime movie.  By placing the "moov" atom near the beginning of the file,
// the movie can start playing almost as fast as it is downloaded.
//
// Package qtfaststart is based on the original qt-faststart tool, written in C
// by Mike Melanson and now distributed as part of the FFmpeg project.  It also
// draws inspiration from Daniel Taylor's Python implementation of that utility.
package qtfaststart
