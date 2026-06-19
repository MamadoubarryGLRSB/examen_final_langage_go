# JOURNAL_IA.md

## Comment j'ai utilisé Cursor

Cursor m'a aidé surtout sur l'architecture du projet : découpage en packages, placement des interfaces `Checker` et `Store`, et le code répétitif (struct tags JSON, handlers de base, middleware de logging, squelette des tests).

## Ce que j'ai fait sans l'IA

Pour l'API REST, la gestion des erreurs (`errors.Is`, `errors.As`), le worker pool et les channels, j'ai surtout regardé la doc officielle Go et mes notes de cours. J'ai relu le code généré pour vérifier que la fermeture des channels, les timeouts et le format JSON correspondaient au sujet.

## Ce que j'ai ajusté

L'ordre de fermeture des channels dans le pool, le contrat JSON exact, et les tests (validation, 404, annulation par context). J'ai aussi corrigé un data race dans un test de concurrence.

## Bilan

L'IA m'a fait gagner du temps sur la structure et le boilerplate. Le reste je l'ai compris et vérifié avec la doc et le cours.
