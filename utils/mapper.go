package utils

import (
	"github.com/pkg/errors"
	"golang.org/x/mod/module"
	"strings"
)

func MapPath(path string) (string, string, error) {
	// not sure if this is required...
	mappings := map[string]string{
		"google.golang.org/grpc":      "github.com/grpc/grpc-go",
		"google.golang.org/protobuf":  "github.com/protocolbuffers/protobuf-go",
		"google.golang.org/api":       "github.com/googleapis/google-api-go-client",
		"google.golang.org/genproto":  "github.com/googleapis/go-genproto",
		"go.opentelemetry.io/proto":   "github.com/open-telemetry/opentelemetry-proto-go",
		"google.golang.org/appengine": "github.com/golang/appengine",
		"honnef.co/go/tools":          "github.com/dominikh/go-tools",
		"golang.org/x":                "github.com/golang",
		"cloud.google.com/go":         "github.com/googleapis/google-cloud-go",
		"go.mongodb.org/mongo-driver": "github.com/mongodb/mongo-go-driver",
		"go.uber.org":                 "github.com/uber-go",
		"go.opencensus.io":            "github.com/census-instrumentation/opencensus-go",
		"gotest.tools":                "github.com/gotestyourself/gotest.tools",
		"bazil.org/fuse":              "github.com/bazil/fuse",
		"go.etcd.io/bbolt":            "github.com/etcd-io/bbolt",
		"rsc.io":                      "github.com/rsc",
		"k8s.io":                      "github.com/kubernetes",
		"go.etcd.io":                  "github.com/etcd-io",
		"sigs.k8s.io":                 "github.com/kubernetes-sigs",
	}

	path, major, _ := module.SplitPathVersion(path)
	path = strings.TrimPrefix(path, "/")
	major = strings.TrimPrefix(major, ".")
	major = strings.TrimPrefix(major, "/")

	if strings.HasPrefix(path, "gopkg.in/") {
		switch parts := strings.Split(path, "/"); len(parts) {
		case 2:
			path = parts[0] + "/" + "go-" + parts[1] + "/" + parts[1]
		case 3:
			break
		default:
			return "", "", errors.Errorf("invalid gopkg.in path '%s': unexpected number of '/'", path)
		}
		path = strings.Replace(path, "gopkg.in/", "github.com/", 1)
	} else {
		for key, value := range mappings {
			if strings.HasPrefix(path, key) {
				path = strings.Replace(path, key, value, 1)
			}
		}
	}

	if major == "v1" || major == "v0" {
		major = ""
	}

	return path, major, nil
}
