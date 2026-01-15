package combinator

type Processor struct {
	rdbMap map[string]RDB
	kvMap  map[string]KV
}

func NewProcessor() *Processor {
	return &Processor{
		rdbMap: make(map[string]RDB),
		kvMap:  make(map[string]KV),
	}
}

func (p *Processor) AddRDB(id string, rdb RDB) {
	p.rdbMap[id] = rdb
}

func (p *Processor) AddKV(id string, kv KV) {
	p.kvMap[id] = kv
}

func (p *Processor) GetRDB(id string) (RDB, bool) {
	rdb, exists := p.rdbMap[id]
	return rdb, exists
}

func (p *Processor) GetKV(id string) (KV, bool) {
	kv, exists := p.kvMap[id]
	return kv, exists
}

func (p *Processor) Start() error {
	for _, rdb := range p.rdbMap {
		if err := rdb.Start(); err != nil {
			return err
		}
	}
	return nil
}
