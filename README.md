# GlobalStateSnapshot
Chandy-Lamport algorithm for snapshotting the global state of a distributed system


# Setup
- File `github.com/DistributedClocks/GoVector/govec/govec.go` : replace line 431 for 
print real timestamps on nanoseconds to readable format
>- Original content: `buffer.WriteString(strconv.FormatInt(time.Now().UnixNano(), 10))`  
>- New time printer: `dt := time.Now()`   
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
`buffer.WriteString(dt.Format(time.StampMilli))` 

- If you get an error related with arguments number for `dec.Decode()` at compiling time, 
you must also replace the line 209 of `govec.go` file: `err = dec.Decode(&d.Pid, &d.Payload, &d.VcMap)` &rarr;
 `err = dec.DecodeMulti(&d.Pid, &d.Payload, &d.VcMap)`    
This error is due to the fact that the used package `github.com/vmihailenco/msgpack` is newer than the one
supported by the GoVector package.  


## TODO

- [X] Añadir RPC
- [X] Añadir retardos
- [X] Print estado global
- [X] Bloqueos al hacer snapshot, era por la sincronización de channels concurrentes no buferrizads
- [X] Recibe msg nil
- [X] Error al enviar un msg
- [X] Añadir prerecording msg

- [ ] Pub key via ssh connection
- [X] Test en local y verificar funcionamiento
- [ ] Pruebas en distribuidos
- [ ] Memoria
- [ ] Limpiar Código
