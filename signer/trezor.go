package signer

type TrezorSigner struct {
}

func NewTrezorSigner(data *SignerData) (TrezorSigner, error) {

	t := TrezorSigner{}

	return t, nil
}
