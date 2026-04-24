package dlq

// NOTE: if the public fields of [Envelope] type change, then the `#Affix()`, `#isDefault()`, `#mergeOver()`, `#prepare()` method will need to be adjusted accordingly

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"time"

	d "github.com/synesissoftware/Diagnosticism.Go"

	snx_lib_net_ip "github.com/Fenway-snx/synthetix-mcp/internal/lib/net/ip"
	snx_lib_utils_build "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/build"
	snx_lib_utils_marshal "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/marshal"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

type envelopeInternals struct {
	ByteOrder        string   `json:"byte_order"`         // human-readable string representing the byte order of the machine that posted the letter
	Commit           string   `json:"commit"`             // commit id, if available, of the binary at the time of building
	GoroutineCount   int      `json:"goroutine_count"`    // number of goroutines in the process that posted the letter
	FileLineFunction string   `json:"file_line_function"` // the file+line+func of the call context that posted the letter
	HostName         string   `json:"host_name"`          // the hostname of the machine on which the letter was posted
	IPAddresses      []string `json:"ip_addresses"`       // ipAddress(es) of the machine on which the letter was posted
	PId              int      `json:"pid"`                // OS id of the process that posted the letter
	ProcessName      string   `json:"process_name"`       // OS name of the process that posted the letter

	Letter struct {
		String                      string `json:"string_form"`
		JSONConversionWasIncomplete bool   `json:"json_conversion_was_incomplete"`
	} `json:"letter"` // _the_ letter
}

// Envelope type.
type Envelope struct {
	Application string    `json:"application"`     // Optional application name for the program or service that posted the letter
	Error       string    `json:"error,omitempty"` // Optional originating error message from the client code that posted the letter
	System      string    `json:"system"`          // Optional system name for the system that posted the letter
	Subsystem   string    `json:"subsystem"`       // Optional subsystem name for the subsystem that posted the letter
	PostTime    time.Time `json:"post_time"`       // Optional event time for the post, which will otherwise be inferred when posting is attempted

	envelopeInternals
}

func (e *Envelope) LetterString() (letterString string, jsonConversionWasIncomplete bool) {

	letterString = e.envelopeInternals.Letter.String
	jsonConversionWasIncomplete = e.envelopeInternals.Letter.JSONConversionWasIncomplete

	return
}

func (e *Envelope) ByteOrder() string {
	return e.envelopeInternals.ByteOrder
}

func (e *Envelope) Commit() string {
	return e.envelopeInternals.Commit
}

func (e *Envelope) FileLineFunction() string {
	return e.envelopeInternals.FileLineFunction
}

func (e *Envelope) GoroutineCount() int {
	return e.envelopeInternals.GoroutineCount
}

func (e *Envelope) HostName() string {
	return e.envelopeInternals.HostName
}

func (e *Envelope) IPAddresses() []string {
	return e.envelopeInternals.IPAddresses
}

func (e *Envelope) PId() int {
	return e.envelopeInternals.PId
}

func (e *Envelope) ProcessName() string {
	return e.envelopeInternals.ProcessName
}

// Populates diagnostic fields on the envelope within the call
// context. When cached is non-nil, process-invariant fields are
// stamped from the cache; otherwise they are computed on the spot.
//
// Idempotent: fields already set to non-zero values are never
// overwritten, so calling affix more than once on the same
// envelope is safe and has no additional effect.
func (e *Envelope) affix(depth int, cached *cachedInvariants) {

	// externals

	if e.PostTime.IsZero() {
		e.PostTime = snx_lib_utils_time.Now()
	}

	// process-invariant internals (prefer cached values)

	if e.envelopeInternals.ByteOrder == "" {
		if cached != nil {
			e.envelopeInternals.ByteOrder = cached.byteOrder
		} else {
			e.envelopeInternals.ByteOrder = byteOrder()
		}
	}

	if e.envelopeInternals.Commit == "" {
		if cached != nil {
			e.envelopeInternals.Commit = cached.commit
		} else {
			e.envelopeInternals.Commit = getCommit()
		}
	}

	if e.envelopeInternals.HostName == "" {
		if cached != nil {
			e.envelopeInternals.HostName = cached.hostName
		} else {
			e.envelopeInternals.HostName, _ = os.Hostname()
		}
	}

	if len(e.envelopeInternals.IPAddresses) == 0 {
		if cached != nil {
			e.envelopeInternals.IPAddresses = cached.ipAddresses
		} else {
			e.envelopeInternals.IPAddresses = bestIPAddresses()
		}
	}

	if e.envelopeInternals.PId == 0 {
		if cached != nil {
			e.envelopeInternals.PId = cached.pId
		} else {
			e.envelopeInternals.PId = os.Getpid()
		}
	}

	if e.envelopeInternals.ProcessName == "" {
		if cached != nil {
			e.envelopeInternals.ProcessName = cached.processName
		} else {
			e.envelopeInternals.ProcessName = filepath.Base(os.Args[0])
		}
	}

	// per-post internals (always computed fresh)

	e.affixFileLineFunction(depth + 1)

	if e.envelopeInternals.GoroutineCount == 0 {
		e.envelopeInternals.GoroutineCount = runtime.NumGoroutine()
	}
}

// Sets only the file+line+function field on the envelope within the call
// context. Like affix, this is idempotent: if the field is already set, no
// work is done.
func (e *Envelope) affixFileLineFunction(depth int) {
	if e.envelopeInternals.FileLineFunction == "" {
		e.envelopeInternals.FileLineFunction, _ = d.GetFileLineFunctionFor(1 + depth)
	}
}

// Computes the process-invariant values once, for caching in the
// handler.
func computeInvariants() cachedInvariants {
	hostName, _ := os.Hostname()

	return cachedInvariants{
		byteOrder:   byteOrder(),
		commit:      getCommit(),
		hostName:    hostName,
		ipAddresses: bestIPAddresses(),
		pId:         os.Getpid(),
		processName: filepath.Base(os.Args[0]),
	}
}

// -------------------------------------
// - (private) implementation methods;
// -------------------------------------

// Non-mutating check for whether the given instance has any
// explicitly-specified public fields.
func (e *Envelope) isDefault() bool {

	if e.Application != "" {
		return false
	}
	if e.Error != "" {
		return false
	}
	if e.System != "" {
		return false
	}
	if e.Subsystem != "" {
		return false
	}
	if !e.PostTime.IsZero() {
		return false
	}

	return true
}

// Merges the public elements of lhs and rhs into a result envelope.
// Non-empty rhs fields unconditionally override the corresponding lhs
// fields; lhs fields survive only where rhs contains zero values.
//
// This is intentional: rhs represents an authoritative envelope (e.g.
// service identity set at bootstrap) whose fields must not be overridden by
// per-call values. It is not a "fill gaps with defaults" merge.
//
// The private contents of lhs are copied to result exactly; the private
// contents of rhs are ignored.
//
// Warn:
// Results are undefined if lhs and rhs are the same instance.
func (lhs Envelope) mergeOver(rhs Envelope) (result Envelope) {

	result = lhs

	// Non-empty rhs fields unconditionally win

	if rhs.Application != "" {
		result.Application = rhs.Application
	}
	if rhs.Error != "" {
		result.Error = rhs.Error
	}
	if rhs.System != "" {
		result.System = rhs.System
	}
	if rhs.Subsystem != "" {
		result.Subsystem = rhs.Subsystem
	}
	if !rhs.PostTime.IsZero() {
		result.PostTime = rhs.PostTime
	}

	return
}

func (e Envelope) prepare(letter any) (envelopeJSONString string, err error) {
	var bytes []byte
	var jsonConversionWasIncomplete bool

	bytes, err = json.Marshal(letter)
	if err != nil {
		jsonConversionWasIncomplete = true
		bytes, err = snx_lib_utils_marshal.SafeMarshalJSON(letter)
	}
	if err != nil {
		return
	}

	letterString := string(bytes)

	e.envelopeInternals.Letter.String = letterString
	e.envelopeInternals.Letter.JSONConversionWasIncomplete = jsonConversionWasIncomplete

	// It is valid to assume that `envelope` will pass `json.Marshal()`
	// because it will only ever be called once the letter has been
	// converted.
	//
	// However, we will check defensively since DLQ is the last line of
	// defense and this function might change in the future.

	bytes, err = json.Marshal(e)
	if err != nil {
		return
	}

	envelopeJSONString = string(bytes)

	return
}

// -------------------------------------
// - (private) helper functions;
// -------------------------------------

func bestIPAddresses() []string {
	r, _ := snx_lib_net_ip.GetHostIPAddresses()

	return r
}

func byteOrder() string {
	var buf [4]byte
	binary.NativeEndian.PutUint32(buf[:], 0x01020304)
	switch buf[0] {
	case 0x01:
		return "big-endian"
	case 0x04:
		return "little-endian"
	default:
		return "unknown"
	}
}

func getCommit() string {
	return snx_lib_utils_build.BuildCommit()
}
