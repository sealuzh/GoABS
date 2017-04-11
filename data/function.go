package data

// Function represents a Go function.
// It does not contain function parameters nor return types, as they are not part of the function signature.
// The method receiver is part of the signature (if available).
type Function struct {
	Path     string
	File     string
	Name     string
	Receiver string
}

type File []Function

type FileMap map[string]File

type PackageMap map[string]FileMap
