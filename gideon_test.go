package main

import (
    "fmt"
    "testing"
    "github.com/pelletier/go-toml"
)



func TestHello(t *testing.T) {
    fmt.Println("Gideon test runner")
}

func TestDefineWorkload(t *testing.T) {
    content := `
        [workloads]
            set = 0
            get = 0
            delete = 0
            update = 0
            expire = 0 
            ttl = 15
            templates = ["templates.", "umerdom"]
            opRate = 100
    `
    wTree, err := toml.Load(content)
    if err != nil {
        t.Fatalf(err.Error())
    }
    if !wTree.Has(WORKLOADS) {
        t.Fatalf("toml parser broken")
    }

    spec := wTree.Get(WORKLOADS).(*toml.TomlTree)
    w := new(Workload)
    UpdateTypeWithSpec(w, spec)
    if  w.Set != 0 || w.Get != 0 ||
       w.Ttl != 15 || w.OpRate != 100 {
       t.FailNow()
    }
    if tmpl := w.Templates[0]; tmpl != "templates." {
       t.FailNow()
    }
}


func TestOverrideWorkload(t *testing.T) {
    content := `
        [workloads]
            set = 0 
    `
    wTree, _:= toml.Load(content)
    spec := wTree.Get(WORKLOADS).(*toml.TomlTree)
    w := new(Workload)
    UpdateTypeWithSpec(w, spec)
    if w.Set != 0 {
        t.Errorf("%d != %d", w.Set, 0)
    }

    content = `
        [workloads]
            set = 9 
    `
    wTree, _ = toml.Load(content)
    spec = wTree.Get(WORKLOADS).(*toml.TomlTree)
    UpdateTypeWithSpec(w, spec)
    if w.Set != 9 {
        t.Errorf("%d != %d", w.Set, 9)
    }

}


func TestDefinePhase(t *testing.T) {
    content := `
        [phases]
            workloads = [
              ["workloads.", "buckets.*", "conditions."]
            ]

            add = [] 
            remove = [] 
            autoFailover = []
            runtime = 30
    `
    pTree, err := toml.Load(content)
    if err != nil {
        t.Fatalf(err.Error())
    }

    spec := pTree.Get(PHASES).(*toml.TomlTree)
    phase := new(Phase)
    UpdateTypeWithSpec(phase, spec)

    if phase.Workloads[0][0] != "workloads." ||
       phase.Workloads[0][1] != "buckets.*"   ||
       phase.Workloads[0][2] != "conditions." {
        t.Errorf("failed to set phase workloads")
    }
}


func TestOverridePhase(t *testing.T) {
    content := `
        [phases]
            workloads = [
              ["workloads.", "buckets.*", "conditions."]
            ]

            add = [] 
            remove = [] 
            autoFailover = []
            runtime = 30
    `
    pTree, _ := toml.Load(content)
    spec := pTree.Get(PHASES).(*toml.TomlTree)
    phase := new(Phase)
    UpdateTypeWithSpec(phase, spec)

    content = `
        [phases]
            workloads = [
              ["SetOnly"]
            ]
    `
    pTree, _ = toml.Load(content)
    spec = pTree.Get(PHASES).(*toml.TomlTree)
    UpdateTypeWithSpec(phase, spec)

    if phase.Workloads[0][0] != "SetOnly"  {
        t.Errorf("failed to update phase workloads")
    }

}
