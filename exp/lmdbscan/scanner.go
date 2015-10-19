/*
Package lmdbscan provides a wrapper for lmdb.Cursor to simplify iteration.
This package is experimental and it's API may change.
*/
package lmdbscan

import (
	"fmt"

	"github.com/bmatsuo/lmdb-go/lmdb"
)

// Scanner is a low level construct for scanning databases inside a
// transaction.
type Scanner struct {
	dbi     lmdb.DBI
	dbflags uint
	txn     *lmdb.Txn
	cur     *lmdb.Cursor
	op      uint
	setop   *uint
	setkey  []byte
	setval  []byte
	key     []byte
	val     []byte
	err     error
}

// New allocates and intializes a Scanner for dbi within txn.
func New(txn *lmdb.Txn, dbi lmdb.DBI) *Scanner {
	s := &Scanner{
		dbi: dbi,
		txn: txn,
	}
	if dbi != 0 {
		s.dbflags, s.err = txn.Flags(dbi)
		if s.err != nil {
			return s
		}
	}
	s.op = lmdb.Next

	s.cur, s.err = txn.OpenCursor(dbi)
	return s
}

// Cursor returns the lmdb.Cursor underlying s.  Cursor returns nil if the
// scanner is closed.
func (s *Scanner) Cursor() *lmdb.Cursor {
	return s.cur
}

// Del will delete the key at the current cursor location.
//
// Del is deprecated.  Instead use s.Cursor().Del(flags).
func (s *Scanner) Del(flags uint) error {
	if s.cur == nil {
		return fmt.Errorf("scanner is closed")
	}
	return s.cur.Del(flags)
}

// Key returns the key read during the last call to Scan.
func (s *Scanner) Key() []byte {
	return s.key
}

// Val returns the value read during the last call to Scan.
func (s *Scanner) Val() []byte {
	return s.val
}

// Set marks the starting position for iteration.  On the next call to s.Scan()
// the underlying cursor will be moved as
//		c.Get(k, v, op)
func (s *Scanner) Set(k, v []byte, op uint) {
	if s.err != nil {
		return
	}
	s.setop = new(uint)
	*s.setop = op
	s.setkey = k
	s.setval = v
}

// SetNext determines the cursor behavior for subsequent calls to s.Scan().
// The immediately following call to s.Scan() behaves as if s.Set(k,v,opset)
// was called.  Subsequent calls move the cursor as
//		c.Get(nil, nil, opnext)
func (s *Scanner) SetNext(k, v []byte, opset, opnext uint) {
	s.Set(k, v, opset)
	s.op = opnext
}

// Scan gets key-value successive pairs with the underlying cursor until one
// matches the supplied filters.  If all filters return a nil error for the
// current pair, true is returned.  Scan returns false if all key-value pairs
// where exhausted.
func (s *Scanner) Scan() bool {
	if s.setop == nil {
		s.key, s.val, s.err = s.cur.Get(nil, nil, s.op)
	} else {
		s.key, s.val, s.err = s.cur.Get(s.setkey, s.setval, *s.setop)
		s.setkey = nil
		s.setval = nil
		s.setop = nil
	}
	return s.err == nil
}

// Err returns a non-nil error if and only if the previous call to s.Scan()
// resulted in an error other than lmdb.ErrNotFound.
func (s *Scanner) Err() error {
	if lmdb.IsNotFound(s.err) {
		return nil
	}
	return s.err
}

// Close closes the cursor underlying s and clears its ows internal structures.
// Close does not attempt to terminate the enclosing transaction.
//
// Scan must not be called after Close.
func (s *Scanner) Close() {
	s.txn = nil
	if s.cur != nil {
		s.cur.Close()
		s.cur = nil
	}
}
