//nolint:staticcheck
package types

import golangproto "github.com/golang/protobuf/proto"

// grpc-gateway v1 only resolves enum names that golang/protobuf has registered.
func init() {
	golangproto.RegisterEnum("nvnmchain.anchoring.v1.RegistryNameMatchMode", RegistryNameMatchMode_name, RegistryNameMatchMode_value)
}
