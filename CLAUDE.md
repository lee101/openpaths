don't use plan mode, just manually write docs into a plan/ folder and then we can.
Thoroughly implement those instead of using the actual plan mode be thorough

## Prod
- Server: `ssh -o StrictHostKeyChecking=no administrator@93.127.141.100` (alias: `openpaths-prod`)
- Remote dir: `/nvme0n1-disk/code/openpaths`
- Deploy: `./deploy.sh {site|api|env|setup|all}` -- see [deploy.md](deploy.md)
- DB & server details: see [servers.md](servers.md) (gitignored)
