package main


import (
    "fmt"
    "log"
    "strings"
    "reflect"
    "errors"
    "github.com/pelletier/go-toml"
)


const (
    TEST        = string("test")
    WORKLOADS   = string("workloads")
    BUCKETS     = string("buckets")
    NODES       = string("nodes")
    CONDITIONS  = string("conditions")
    PHASES      = string("phases")
    SET         = string("set")
)

type Tasker interface {
    Run()
}

type Workload struct {

    Set int64
    Get int64
    Delete int64
    Update int64
    Expire int64
    Ttl int64
    OpRate int64
    Templates []string

}

func (w *Workload) Run() {
    fmt.Println(w.Set)
}

type Phase struct {
    tasks map[string]Tasker
    Workloads [][]string
    Add []string
    Remove []string
    AutoFailover []string
    Runtime int64
}


func (p *Phase) AddWorkloads(workloads []interface{}, t *Test){

    for _, w := range workloads {
        wDef := w.([]interface{})
        for i, v := range wDef {
            if i == 0 {
                s := []string{WORKLOADS, v.(string)}
                wPath := strings.Join(s, ".")
                if !t.spec.Has(wPath) {
                    log.Fatalf("missing defn for %s", wPath)
                }

                // set to base workload
                workload := t.defaults.workload

                // update with values from spec
                wTree := t.spec.Get(wPath).(*toml.TomlTree)
                UpdateTypeWithSpec(workload, wTree)

                // add to phase
                p.tasks[wPath] = Tasker(workload)
            }
        }
    }

}

func (p *Phase) Run() {
    for _, task := range p.tasks {
        task.Run()
    }
}

type Test struct {
    name string
    phases map[string]*Phase
    spec *toml.TomlTree
    defaults *DefaultConfig
    base *toml.TomlTree
}

func (t *Test)BuildSubDirective(name string) (interface{}, error){

    // creates inherited directives from test spec

    var err error
    var rc interface{}

    path := strings.Split(name, ".")

    // initialize base object
    switch path[0] {

        case WORKLOADS:
            rc = new(Workload)
            spec :=  t.base.Get(WORKLOADS).(*toml.TomlTree)
            UpdateTypeWithSpec(rc, spec)
        case PHASES:
            rc = new(Phase)
            rc.(*Phase).tasks = make(map[string]Tasker)
            spec :=  t.base.Get(PHASES).(*toml.TomlTree)
            UpdateTypeWithSpec(rc, spec)

    }

    // construct workload hierarchy from test spec
    if len(path) < 2 {
        err = errors.New("Invalid sub-directive "+name)
        return nil, err
    }

    for i := 2; i<=len(path); i++ {
        dName := strings.Join(path[0:i], ".")
        if !t.spec.Has(dName+".") {
            err = errors.New("missing directive "+dName)
        } else {
            subSpec :=  t.spec.Get(dName).(*toml.TomlTree)
            UpdateTypeWithSpec(rc, subSpec)
        }
    }

    return rc, err


}

func (t *Test) LinkPhaseTasks(phase *Phase) error {

    // link phase construct to runnable tasks
    for _, wDef := range phase.Workloads {
        for i, wAttr := range wDef {
            switch i {
                case 0:

                    // link workload directive
                    wName := WORKLOADS+"."+wAttr
                    w, err := t.BuildSubDirective(wName)
                    if err != nil {
                        return err
                    }
                    phase.tasks[wName] = Tasker(w.(*Workload))
            }
        }
    }

    return nil
}

func (t* Test) LinkTestPhases() error {

    var err error
    tDef := t.spec.Get(TEST).(*toml.TomlTree)
    if tDef == nil {
        err = errors.New("Test spec missing [test] directive ")
        return err
    }

    pNames := tDef.Get(PHASES).([]interface{})
    for _, pName := range pNames {
        phase, e := t.BuildSubDirective(pName.(string))
        err = e
        p := phase.(*Phase)
        if err == nil {
            err = t.LinkPhaseTasks(p)
            if err == nil {
                t.phases[pName.(string)] = p
            }
        }
    }


    return err
}


func newTest(spec string) *Test {

    baseSpec, err := toml.LoadFile("def.toml")
    mf(err, "load..def.toml")

    testSpec, err := toml.LoadFile(spec)
    mf(err, "load.."+spec)

    test := new(Test)
    test.spec = testSpec
    test.base = baseSpec
    test.defaults = newDefaultConfig("def.toml")
    test.phases = make(map[string]*Phase)

    return test
}


func (t *Test) Run() error {
    err := t.LinkTestPhases()
    if err != nil {
        return err
    }
    for _, p := range t.phases {
        p.Run()
    }

    return err
}



type DefaultConfig struct {
    workload *Workload
    phase *Phase
}

func (cfg *DefaultConfig) DefineWorkload(wSpec *toml.TomlTree) {

    cfg.workload = new(Workload)
    UpdateTypeWithSpec(cfg.workload, wSpec)

}

func newDefaultConfig (fileName string) *DefaultConfig {

    var config *DefaultConfig
    tomlConfig, err := toml.LoadFile(fileName)
    mf(err, "load_definitions")

    config = new(DefaultConfig)
    for _, key := range tomlConfig.Keys() {

        switch key {
            case WORKLOADS:
                spec := tomlConfig.Get(WORKLOADS).(*toml.TomlTree)
                config.workload = new(Workload)
                UpdateTypeWithSpec(config.workload, spec)

            case PHASES:
                spec := tomlConfig.Get(PHASES).(*toml.TomlTree)
                config.phase = new(Phase)
                UpdateTypeWithSpec(config.phase, spec)
        }
    }

    return config
 }

func UpdateTypeWithSpec(t interface{}, wTree *toml.TomlTree){

    s := reflect.ValueOf(t).Elem()
    for _, key := range wTree.Keys() {

            // get override field name
            fieldName := strings.ToUpper(string(key[0]))+key[1:]
            field := s.FieldByName(fieldName)
            if !field.IsValid() {
                fmt.Println("field %s is invalid..skipping", key)
                continue
            }

            // set underlying spec value with appropriate type
            switch field.Kind() {

                case reflect.Int64:
                    val := wTree.Get(key).(int64)
                    field.SetInt(val)

               case reflect.String:
                    val := wTree.Get(key).(string)
                    field.SetString(val)

               case reflect.Slice:
                    vals := wTree.Get(key).([]interface{})
                    vSlice := makeValueSlice(vals)
                    if vSlice.IsValid() {
                        field.Set(vSlice)
                    }
            }

    }

}

func makeValueSlice(vals []interface{}) reflect.Value {

    var vSlice reflect.Value


    // unpack slice of strings
    for i := range vals {

        // check if value is a subslice
        switch vals[i].(type) {

            case string:
                tp := reflect.TypeOf([]string{""})
                if !vSlice.IsValid() {
                    vSlice = reflect.MakeSlice(tp, len(vals), len(vals))
                }

                // create temp struct Value
                v := reflect.New(reflect.TypeOf(""))
                iv := reflect.Indirect(v)

                // set Value in slice from incomming string
                iv.SetString(vals[i].(string))
                vSlice.Index(i).Set(iv)

            case []interface{}:
                // handle 2d array fields
                tp := reflect.TypeOf([][]string{{""}, {""}})
                if !vSlice.IsValid() {
                    vSlice = reflect.MakeSlice(tp, len(vals), len(vals))
                }
                subSlice := vals[i].([]interface{})
                vSlice.Index(i).Set(makeValueSlice(subSlice))

       }
    }

    return vSlice
}

func mf(err error, msg string) {
    if err != nil {
        log.Fatalf("%v: %v", msg, err)
    }
}

func main() {


    // load test spec
    test := newTest("examples/simple.toml")

    // run
    test.Run()


}
