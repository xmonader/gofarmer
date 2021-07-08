package main

type identityManager struct {
	key KeyPair
}

// Sign signs the message with privateKey and returns a signature.
func (d *identityManager) Sign(message []byte) ([]byte, error) {
	return Sign(d.key.PrivateKey, message)
}

// Verify reports whether sig is a valid signature of message by publicKey.
func (d *identityManager) Verify(message, sig []byte) error {
	return Verify(d.key.PublicKey, message, sig)
}

// Encrypt encrypts message with the public key of the node
func (d *identityManager) Encrypt(message []byte) ([]byte, error) {
	return Encrypt(message, d.key.PublicKey)
}

// Decrypt decrypts message with the private of the node
func (d *identityManager) Decrypt(message []byte) ([]byte, error) {
	return Decrypt(message, d.key.PrivateKey)
}

// EncryptECDH encrypt msg using AES with shared key derived from private key of the node and public key of the other party using Elliptic curve Diffie Helman algorithm
// the nonce if prepended to the encrypted message
func (d *identityManager) EncryptECDH(msg []byte, pk []byte) ([]byte, error) {
	return EncryptECDH(msg, d.key.PrivateKey, pk)
}

// DecryptECDH decrypt AES encrypted msg using a shared key derived from private key of the node and public key of the other party using Elliptic curve Diffie Helman algorithm
func (d *identityManager) DecryptECDH(msg []byte, pk []byte) ([]byte, error) {
	return DecryptECDH(msg, d.key.PrivateKey, pk)
}

// PrivateKey returns the private key of the node
func (d *identityManager) PrivateKey() []byte {
	return d.key.PrivateKey
}
