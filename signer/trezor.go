package signer

type TrezorSigner struct {
}

func NewTrezorSigner(data *Signer) (TrezorSigner, error) {

	t := TrezorSigner{}

	return t, nil
}
