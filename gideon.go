package main


import (
    "fmt"
    "log"
    "strings"
    "reflect"
    "github.com/pelletier/go-toml"
)


const (
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
}


func (t *Test) AddPhase(pTree *toml.TomlTree) {
    phase := new(Phase)
    phase.tasks = make(map[string]Tasker)

    for _, task := range pTree.Keys() {
        switch  task {
            case WORKLOADS:
                workloads := pTree.Get(WORKLOADS).([]interface{})
                phase.AddWorkloads(workloads, t)
                t.phases[pTree.ToString()] = phase
        }
        fmt.Println(task)
    }
}

func newTest(spec string, config *DefaultConfig) *Test {

    testSpec, err := toml.LoadFile(spec)
    mf(err, "load-spec")

    test := new(Test)
    test.spec = testSpec
    test.defaults = config
    test.phases = make(map[string]*Phase)

    phases := testSpec.Get("test.phases").([]interface{})
    for _,k := range phases {
        phaseName := testSpec.Get(k.(string))
        if phaseName == nil {
            log.Fatalf("missing defined phase %s", k)
        } else {
            phaseTree := phaseName.(*toml.TomlTree)
            test.AddPhase(phaseTree)
        }
    }

    return test
}



func (t *Test) Run() {
    for _, p := range t.phases {
        p.Run()
    }
}





type DefaultConfig struct {
    workload *Workload
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
    workload := tomlConfig.Get(WORKLOADS).(*toml.TomlTree)
    config.DefineWorkload(workload)

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


    // load base definitions
    config := newDefaultConfig("def.toml")

    // load test spec 
    test := newTest("examples/simple.toml", config)

    // run
    test.Run()


}
