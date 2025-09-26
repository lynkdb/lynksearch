package lynksearch

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/lynkdb/lynkapi/go/lynkapi"
)

type InstanceConfig struct {
	// Dir specifies the directory to store index data.
	Dir string `json:"dir" toml:"dir" yaml:"dir"`

	// Spec *lynkapi.TableSpec `json:"spec" toml:"spec" yaml:"spec"`
}

type Instance interface {
	AddDocument(id string, doc map[string]any) error
	Search(q string) *lynkapi.DataResult
	Flush() error
	Close() error
}

func NewInstance(cfg InstanceConfig, spec *lynkapi.TableSpec) (Instance, error) {

	ins := &instance{
		indexer: &indexer{
			tokenIndex:    map[string]*tokenEntry{},
			tokenIntIndex: map[uint32]*tokenEntry{},
			docIndex:      map[string]*docMetaEntry{},
			docIntIndex:   map[uint32]*docMetaEntry{},
		},
	}

	{
		if cfg.Dir == "" {
			return nil, fmt.Errorf("config: dir not setup")
		}

		cfg.Dir = filepath.Clean(cfg.Dir)

		if info, err := os.Stat(cfg.Dir); err != nil {
			if os.IsNotExist(err) {
				return nil, err
			}
		} else if !info.IsDir() {
			return nil, fmt.Errorf("config: dir (%s) is exists", cfg.Dir)
		}

		ins.cfg = cfg
		ins.indexer.dir = cfg.Dir
	}

	{
		if spec == nil || len(spec.Indexes) == 0 {
			return nil, fmt.Errorf("no indexes found")
		}

		for _, idx := range spec.Indexes {
			if idx.Type != lynkapi.TableSpec_Index_FullTextSearch {
				continue
			}
			fs := strings.Split(idx.Fields, ",")
			for _, name := range fs {
				if !slices.Contains(ins.textFields, name) {
					ins.textFields = append(ins.textFields, name)
				}
			}
		}

		if len(ins.textFields) == 0 {
			return nil, fmt.Errorf("no index (type: fts) found")
		}
	}

	if err := ins.init(); err != nil {
		return nil, err
	}

	return ins, nil
}

type instance struct {
	mu sync.Mutex

	textFields []string

	cfg InstanceConfig

	indexer *indexer
}

func (it *instance) init() error {
	return nil
}

func (it *instance) AddDocument(id string, doc map[string]any) error {

	if len(doc) == 0 {
		return nil
	}

	for k, v := range doc {
		k2 := strings.ToLower(k)
		if k != k2 {
			doc[k2] = v
		}

	}

	indexTextFields := []string{}
	for _, name := range it.textFields {
		v, ok := doc[name]
		if !ok || v == nil {
			continue
		}
		switch v.(type) {
		case string:
			indexTextFields = append(indexTextFields, v.(string))
		}
	}

	if len(indexTextFields) == 0 {
		return nil
	}

	it.mu.Lock()
	defer it.mu.Unlock()

	return it.indexer.index(id, indexTextFields)
}

func (it *instance) Search(q string) *lynkapi.DataResult {
	return it.indexer.search(q)
}

func (it *instance) Flush() error {
	return it.indexer.flush()
}

func (it *instance) Close() error {
	return nil
}
