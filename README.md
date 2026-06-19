# URLWatch — Examen final Langage Go

Microservice de vérification d'URLs en masse. Un client envoie un lot d'URLs, le service les vérifie en parallèle (code HTTP, latence, disponibilité), agrège les résultats et les expose via une API REST.

**Prérequis** : Go 1.22 ou supérieur.

## Build

```bash
go build ./...
go vet ./...
```

## Lancer le serveur

```bash
go run ./cmd/urlwatch
```

Le service écoute sur `http://localhost:8080` par défaut. Aucune configuration n'est obligatoire.

Pour lancer :

```bash
go run ./cmd/urlwatch
```

Si besoin, on peut passer deux variables dans le terminal (pas de secrets, juste de la config locale, voir `.env.example`) :

- `LISTEN_ADDR` pour changer le port (défaut `:8080`)
- `LOG_LEVEL` pour le niveau de log : `debug`, `info`, `warn` ou `error`

Exemple :

```bash
LISTEN_ADDR=:9090 LOG_LEVEL=debug go run ./cmd/urlwatch
```

Aucune clé ni token n'est utilisé dans ce projet.

## Tests

```bash
go test ./...
go test -race ./...
```

## Tester l'API

Les exemples ci-dessous fonctionnent avec Postman ou curl. Le serveur doit être lancé au préalable.

### GET /healthz — Sonde de vivacité

```
GET http://localhost:8080/healthz
```

Réponse attendue : `200` avec `{"status":"ok"}`.

### POST /v1/checks — Créer et exécuter un lot

```
POST http://localhost:8080/v1/checks
Content-Type: application/json
```

```json
{
  "urls": ["https://go.dev", "https://exemple.invalid"],
  "options": {
    "concurrency": 4,
    "timeout_ms": 2000
  }
}
```

Réponse attendue : `201 Created` avec `batch_id`, `summary` et `results`.

Exemple avec curl :

```bash
curl -s -X POST http://localhost:8080/v1/checks \
  -H "Content-Type: application/json" \
  -d '{"urls":["https://go.dev","https://exemple.invalid"],"options":{"concurrency":4,"timeout_ms":2000}}'
```

### GET /v1/checks/{id} — Relire un lot

```
GET http://localhost:8080/v1/checks/b_4f3c1a
```

Remplacer `b_4f3c1a` par le `batch_id` reçu au POST.

Réponse attendue : `200` avec le lot complet, ou `404` si l'identifiant est inconnu.

```bash
curl -s http://localhost:8080/v1/checks/b_inconnu
```

Réponse attendue : `404` avec :

```json
{
  "error": {
    "code": "batch_not_found",
    "message": "aucun lot avec l'id b_inconnu"
  }
}
```

### Erreur de validation

```bash
curl -s -X POST http://localhost:8080/v1/checks \
  -H "Content-Type: application/json" \
  -d '{"urls":[],"options":{"concurrency":4,"timeout_ms":2000}}'
```

Réponse attendue : `400` avec le code `invalid_request`.

## Routes

| Méthode | Route | Description |
|---------|-------|-------------|
| POST | `/v1/checks` | Crée et exécute un lot de vérifications |
| GET | `/v1/checks/{id}` | Récupère un lot existant |
| GET | `/healthz` | Sonde de vivacité |

## Structure du projet

```
cmd/urlwatch/       Point d'entrée, assemblage des dépendances
internal/domain/    Types métier, interfaces, validation
internal/checker/   Vérification HTTP + mock pour les tests
internal/pool/      Worker pool borné (fan-out / fan-in)
internal/store/     Persistance en mémoire
internal/api/       Handlers REST, middleware logging et recovery
```

## Documents complémentaires

- `DESIGN.md` : choix d'architecture et justification technique
- `JOURNAL_IA.md` : usage de l'IA dans ce projet
