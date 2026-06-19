# DESIGN.md

## Découpage

J'ai suivi l'arborescence du sujet. Le package `domain` contient les types et les interfaces `Checker` et `Store`. Les autres packages en dépendent, pas l'inverse.

`checker` fait les vrais appels HTTP. `pool` gère toute la concurrence. `store` sauvegarde les lots en mémoire. `api` expose les routes REST. `main.go` se contente de tout brancher.

Ça permet de tester chaque partie séparément avec des mocks, sans toucher au réseau ni à une vraie base.

## Concurrence

Le cœur du projet est dans `internal/pool/pool.go`.

Je lance un nombre fixe de workers (`options.concurrency`, défaut 8). Chaque worker lit des URLs dans un channel `jobs` et renvoie le résultat dans un channel `results`. Les deux channels sont bufferisés à `len(urls)` pour éviter qu'une goroutine bloque une autre.

Quand toutes les URLs sont envoyées, je ferme `jobs`. Les workers s'arrêtent tout seuls avec `for url := range jobs`. Un `WaitGroup` attend la fin de tous les workers avant de fermer `results`.

Chaque URL a son propre timeout via `context.WithTimeout`. Si le contexte parent est annulé, le worker renvoie une erreur sans faire d'appel réseau.

Une URL qui échoue (DNS, timeout, 500...) ne fait pas planter le lot. Elle apparaît dans `results` avec `ok: false`.

## Goroutines

Le risque principal était un deadlock si `results` se fermait trop tôt ou si un worker restait bloqué. D'où le buffering des channels et la fermeture de `results` seulement après le `WaitGroup`.

Chaque worker appelle `cancel()` après son `context.WithTimeout` pour ne pas garder de contexte ouvert.

## Erreurs

- `ErrBatchNotFound` : lot introuvable, traduit en `404` avec `errors.Is`
- `ValidationError` : champ invalide dans le JSON, traduit en `400` avec `errors.As`
- Erreur réseau sur une URL : ce n'est pas une erreur du service, c'est un résultat normal dans `CheckResult`

## Pourquoi Go pour ce projet

Le sujet demande de la concurrence, un microservice backend et une gestion propre des timeouts. Go colle bien à ça.

1. **Concurrence native** : dans `pool.go`, le worker pool avec fan-out/fan-in tient en une trentaine de lignes avec goroutines, channels et `WaitGroup`. En Java il faudrait `ExecutorService` et `CompletableFuture`. En Python, `asyncio` ou `ThreadPoolExecutor` avec une gestion d'erreurs moins directe.

2. **`context` intégré** : chaque URL a son timeout via `context.WithTimeout`, et l'annulation remonte proprement si le lot est interrompu. C'est exactement ce que le sujet demande, sans lib externe.

3. **`net/http` en stdlib** : avec Go 1.22, le routeur gère `GET /v1/checks/{id}` nativement dans `router.go`. Pas besoin de Gin ou Chi pour trois routes. En Python il faudrait FastAPI ou Flask, en Java Spring Boot avec plus de config.

4. **Un seul binaire** : `go build` produit un exécutable autonome, facile à lancer et à tester en local.

**Limite ressentie** : pour des agrégations plus complexes, Go est un peu plus verbeux qu'en Python ou Rust. Ici `AggregateSummary` avec une `map[bool]int` suffit, mais sur un projet plus gros ça se sentirait.

## Bonus

**SQLite** : `internal/store/sqlite.go` implémente la même interface `Store` avec `database/sql`. J'ai pas pris GORM parce que le schéma est simple (une table, les résultats en JSON). Le driver `modernc.org/sqlite` évite CGO, pratique pour Docker.

**Async** : `POST /v1/checks?async=true` sauvegarde un lot `pending` et renvoie `202`. Le pool tourne en goroutine, puis le lot passe à `done`.

**Liste** : `GET /v1/checks` avec `page`, `limit` et filtre `status`. Implémenté sur memory et sqlite via `Store.List`.

**Arrêt gracieux** : déjà dans `main.go` avec `signal` + `Server.Shutdown`.

**Docker** : Dockerfile multi-stage, binaire statique sans CGO.
