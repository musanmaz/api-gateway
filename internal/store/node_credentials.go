package store

import(
 "context"
 "crypto/sha256"
 "encoding/hex"
 "errors"
 "strings"
 "time"
 "github.com/jackc/pgx/v5"
 "gateway-control/internal/validate"
)
func(s *Store) SetNodeCredential(ctx context.Context,node,hash,actor string)error{
 if !nodeNamePattern.MatchString(node){return &validate.Error{Details:[]string{"invalid node name"}}}
 if len(hash)!=64{return &validate.Error{Details:[]string{"sha256 digest required"}}}
 decoded,err:=hex.DecodeString(strings.ToLower(hash));if err!=nil{return &validate.Error{Details:[]string{"invalid sha256 digest"}}}
 tx,err:=s.pool.Begin(ctx);if err!=nil{return err};defer tx.Rollback(ctx)
 var old []byte
 err=tx.QueryRow(ctx,"SELECT token_sha256 FROM node_agent_credentials WHERE node_name=$1 FOR UPDATE",node).Scan(&old)
 if err!=nil && !errors.Is(err,pgx.ErrNoRows){return err}
 _,err=tx.Exec(ctx,`
 INSERT INTO node_agent_credentials(node_name,token_sha256,rotated_at)
 SELECT name,$2,now() FROM nodes WHERE name=$1
 ON CONFLICT(node_name) DO UPDATE SET token_sha256=EXCLUDED.token_sha256,rotated_at=EXCLUDED.rotated_at`,node,decoded)
 if err!=nil{return err}
 if err=insertAudit(ctx,tx,actor,"node.agent-credential.rotate","node:"+node,
  map[string]any{"wasConfigured":len(old)>0},map[string]any{"configured":true,"when":time.Now().UTC().Format(time.RFC3339)},"ok");err!=nil{return err}
 return tx.Commit(ctx)
}
func(s *Store) AgentCredentialOwner(ctx context.Context,token string)(string,bool){
 if len(token)<32 || len(token)>512{return "",false}
 hash:=sha256.Sum256([]byte(token))
 var node string
 err:=s.pool.QueryRow(ctx,"SELECT node_name FROM node_agent_credentials WHERE token_sha256=$1",hash[:]).Scan(&node)
 return node,err==nil && node!=""
}
