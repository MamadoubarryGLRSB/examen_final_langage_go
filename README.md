# URLWatch — Examen final Langage Go

Microservice de vérification d'URLs en masse. Un client envoie un lot d'URLs, le service les vérifie en parallèle (code HTTP, latence, disponibilité), agrège les résultats et les expose via une API REST.

**Prérequis** : Go 1.22 ou supérieur, Docker (optionnel, pour le bonus).

---

## Guide rapide pour la correction

### 1. Vérifier que le projet compile et que les tests passent

```bash
go build ./...
go vet ./...
go test ./...
go test -race ./...
```

### 2. Lancer le serveur (mode par défaut, mémoire)

Rien à configurer, ça suffit pour tester le socle du sujet :

```bash
go run ./cmd/urlwatch
```

Le service écoute sur `http://localhost:8080`.

### 3. Lancer avec SQLite (bonus persistance)

Dans un autre terminal, une fois le serveur lancé :

```bash
STORE=sqlite SQLITE_PATH=urlwatch.db go run ./cmd/urlwatch
```

Les lots sont sauvegardés dans le fichier `urlwatch.db` à la racine du projet.

### 4. Lancer avec Docker (bonus)

```bash
docker build -t urlwatch .
docker run -p 8080:8080 urlwatch
```

L'image utilise SQLite par défaut. Le service est accessible sur `http://localhost:8080`.

### 5. Tester l'API

Avec le serveur lancé (étape 2, 3 ou 4) :

```bash
# Vivacité
curl http://localhost:8080/healthz

# Créer un lot (réponse 201)
curl -s -X POST http://localhost:8080/v1/checks \
  -H "Content-Type: application/json" \
  -d '{"urls":["https://go.dev","https://exemple.invalid"],"options":{"concurrency":4,"timeout_ms":2000}}'

# Relire un lot (remplacer b_xxxxx par le batch_id reçu)
curl -s http://localhost:8080/v1/checks/b_xxxxx
```

Les exemples détaillés (erreurs 404/400, mode async, liste) sont plus bas.

---

## Variables d'environnement

Toutes optionnelles. Voir aussi `.env.example`.

| Variable | Défaut | Rôle |
|----------|--------|------|
| `LISTEN_ADDR` | `:8080` | Port d'écoute |
| `LOG_LEVEL` | `info` | Niveau de log (`debug`, `info`, `warn`, `error`) |
| `STORE` | `memory` | `memory` ou `sqlite` |
| `SQLITE_PATH` | `urlwatch.db` | Fichier SQLite si `STORE=sqlite` |

Aucun secret (clé, token) n'est utilisé dans ce projet.

---

## Routes

| Méthode | Route | Description |
|---------|-------|-------------|
| POST | `/v1/checks` | Crée et exécute un lot |
| GET | `/v1/checks/{id}` | Récupère un lot par son id |
| GET | `/healthz` | Sonde de vivacité |
| POST | `/v1/checks?async=true` | Lot en arrière-plan, réponse 202 (bonus) |
| GET | `/v1/checks` | Liste des lots avec pagination (bonus) |

---

## Tester l'API en détail

Le serveur doit tourner avant de lancer ces requêtes (Postman ou curl).

### GET /healthz

```
GET http://localhost:8080/healthz
```

Réponse : `200` avec `{"status":"ok"}`.

### POST /v1/checks

```json
{
  "urls": ["https://go.dev", "https://exemple.invalid"],
  "options": { "concurrency": 4, "timeout_ms": 2000 }
}
```

Réponse : `201 Created` avec `batch_id`, `summary` et `results`.

### GET /v1/checks/{id}

Remplacer `{id}` par le `batch_id` du POST.

Réponse : `200` avec le lot, ou `404` si l'id est inconnu.

### Erreur 404

```bash
curl -s http://localhost:8080/v1/checks/b_inconnu
```

Réponse : `404` avec `"code": "batch_not_found"`.

### Erreur 400 (validation)

```bash
curl -s -X POST http://localhost:8080/v1/checks \
  -H "Content-Type: application/json" \
  -d '{"urls":[],"options":{"concurrency":4,"timeout_ms":2000}}'
```

Réponse : `400` avec `"code": "invalid_request"`.

### Bonus : mode asynchrone

```bash
curl -s -X POST "http://localhost:8080/v1/checks?async=true" \
  -H "Content-Type: application/json" \
  -d '{"urls":["https://go.dev"],"options":{"concurrency":2,"timeout_ms":2000}}'
```

Réponse : `202 Accepted` avec `status: pending`. Relancer le GET sur le `batch_id` jusqu'à voir `status: done`.

### Bonus : liste des lots

```
GET http://localhost:8080/v1/checks?page=1&limit=10&status=done
```

Paramètres optionnels : `page`, `limit`, `status` (`pending` ou `done`).

---

## Structure du projet

```
cmd/urlwatch/       Point d'entrée
internal/domain/    Types, interfaces, validation
internal/checker/   Vérification HTTP + mock
internal/pool/      Worker pool (fan-out / fan-in)
internal/store/     Mémoire ou SQLite
internal/api/       Handlers REST, middleware
```

## Documents complémentaires

- `DESIGN.md` : architecture et choix techniques
- `JOURNAL_IA.md` : usage de l'IA
