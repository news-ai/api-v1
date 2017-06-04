# api

API for NewsAI platform. For products:

1. Media List Management (Tabulae)
2. Media Database

In the `/api/` folder:

- Running development: `goapp serve local.yaml`
- Deploying: `goapp deploy` (can be either `dev.yaml` or `prod.yaml`)
- Rollback: `appcfg.py rollback -A newsai-1166 -V 1 api/`

Indexes:

- Update: `appcfg.py update_indexes api/ -A newsai-1166`
- Delete: `appcfg.py vacuum_indexes api/ -A newsai-1166`

Cron:

- `appcfg.py update_cron api/ -A newsai-1166`
