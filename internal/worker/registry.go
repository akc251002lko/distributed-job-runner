package worker
import("context";"encoding/json";"errors";"fmt";"sync")
type Handler func(context.Context,json.RawMessage)error
type Registry struct{mu sync.RWMutex;handlers map[string]Handler}
func NewRegistry()*Registry{return &Registry{handlers:make(map[string]Handler)}}
func(r *Registry)Register(name string,h Handler)error{if name==""||len(name)>128{return errors.New("handler name must contain 1 to 128 characters")};if h==nil{return errors.New("handler must not be nil")};r.mu.Lock();defer r.mu.Unlock();if _,ok:=r.handlers[name];ok{return fmt.Errorf("handler %q is already registered",name)};r.handlers[name]=h;return nil}
func(r *Registry)Execute(ctx context.Context,name string,payload json.RawMessage)(err error){r.mu.RLock();h,ok:=r.handlers[name];r.mu.RUnlock();if !ok{return fmt.Errorf("unsupported job type %q",name)};if err:=ctx.Err();err!=nil{return err};defer func(){if recovered:=recover();recovered!=nil{err=fmt.Errorf("handler panic in %q: %v",name,recovered)}}();return h(ctx,payload)}
func DefaultRegistry()*Registry{r:=NewRegistry();_ = r.Register("noop",func(ctx context.Context,_ json.RawMessage)error{return ctx.Err()});_ = r.Register("echo",func(ctx context.Context,p json.RawMessage)error{if !json.Valid(p){return errors.New("echo payload is not valid JSON")};return ctx.Err()});return r}
