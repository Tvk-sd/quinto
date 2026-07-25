package goatcounter

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// SupportedVersion is the export format quinto understands. GoatCounter
// prefixes the first header cell with it and their docs are explicit that a
// consumer should fail loudly on a change rather than mis-parse into a corrupt
// local database.
const SupportedVersion = 2

// Hit is one row of the export.
//
// Note what is absent: GoatCounter's CSV carries no row identifier. The export
// job reports an overall last_hit_id for use as a cursor, but individual hits
// are anonymous. Key is therefore a content hash — see hitKey.
type Hit struct {
	Key            string
	Path           string
	Title          string
	IsEvent        bool
	Browser        string
	System         string
	Session        string
	Bot            int
	Referrer       string
	ReferrerScheme string
	ScreenSize     string
	Country        string
	FirstVisit     bool
	CreatedAt      string
}

// VersionError reports an export format quinto was not written against.
type VersionError struct {
	Got, Want int
}

func (e *VersionError) Error() string {
	return fmt.Sprintf(
		"GoatCounter export format is version %d, quinto understands %d — "+
			"refusing to parse rather than write corrupt data. Upgrade quinto.",
		e.Got, e.Want)
}

// ParseExport reads a GoatCounter CSV export.
//
// An empty input is not an error: an export with no new hits is a valid, empty
// response and means "nothing since the cursor".
func ParseExport(r io.Reader) ([]Hit, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1 // validated against the header instead

	header, err := cr.Read()
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading export header: %w", err)
	}

	version, err := parseVersion(header[0])
	if err != nil {
		return nil, err
	}
	if version != SupportedVersion {
		return nil, &VersionError{Got: version, Want: SupportedVersion}
	}
	// Strip the version prefix so the first column is addressable by name.
	header[0] = strings.TrimLeft(header[0], "0123456789")

	// Columns are looked up by name, never by position, so a reordered or
	// extended export does not silently shift every field.
	index := make(map[string]int, len(header))
	for i, name := range header {
		index[name] = i
	}
	for _, required := range []string{"Path", "Session", "Date"} {
		if _, ok := index[required]; !ok {
			return nil, fmt.Errorf("export is missing the %q column", required)
		}
	}

	var hits []Hit
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading export row: %w", err)
		}

		get := func(name string) string {
			i, ok := index[name]
			if !ok || i >= len(rec) {
				return ""
			}
			return rec[i]
		}

		hits = append(hits, Hit{
			Key:            hitKey(rec),
			Path:           get("Path"),
			Title:          get("Title"),
			IsEvent:        get("Event") == "1" || strings.EqualFold(get("Event"), "true"),
			Browser:        get("Browser"),
			System:         get("System"),
			Session:        get("Session"),
			Bot:            atoi(get("Bot")),
			Referrer:       get("Referrer"),
			ReferrerScheme: get("Referrer scheme"),
			ScreenSize:     get("Screen size"),
			Country:        get("Location"),
			FirstVisit:     get("FirstVisit") == "1",
			CreatedAt:      get("Date"),
		})
	}
	return hits, nil
}

// parseVersion reads the digits GoatCounter prefixes onto the first header
// cell. Observed live as "2Path" — the docs render it as "2,Path", which is
// misleading: it is one field, not two.
func parseVersion(first string) (int, error) {
	digits := first[:len(first)-len(strings.TrimLeft(first, "0123456789"))]
	if digits == "" {
		return 0, fmt.Errorf("export header %q carries no format version", first)
	}
	return strconv.Atoi(digits)
}

// hitKey identifies a row for deduplication. GoatCounter gives rows no id, so
// the key is a hash of the row itself: re-ingesting the same export is then a
// no-op, and the only rows that collide are ones identical in every field —
// which are indistinguishable anyway.
func hitKey(rec []string) string {
	h := sha256.New()
	for _, f := range rec {
		h.Write([]byte(f))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
