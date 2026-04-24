// This file exists _purely_ to introduce types, constants, variables into
// this package, thereby removing the need for explicit qualification. This
// should only be done for types that are in the same/related layer of
// abstraction as the receiving package.

package doubles

import (
	snx_lib_dlq "github.com/Fenway-snx/synthetix-mcp/internal/lib/dlq"
)

type DeadLetterDeliverer = snx_lib_dlq.DeadLetterDeliverer
type DeadLetterQueue = snx_lib_dlq.DeadLetterQueue
type Envelope = snx_lib_dlq.Envelope
