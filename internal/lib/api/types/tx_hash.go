package types

type TxHash = string

func TxHashFromStringUnvalidated(
	s string,
) TxHash {
	return TxHash(s)
}

func TxHashPtrFromStringPtrUnvalidated(
	p *string,
) *TxHash {
	if p == nil {
		return nil
	} else {
		h := TxHash(*p)

		return &h
	}
}
