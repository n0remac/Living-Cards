# Generated card backgrounds

`go run ./cmd/cardimages` writes generated card-background images and their
JSON manifests here.

Use `make card-images-game ARGS="-seed 42"` from the project root to generate
one stable, card-ID-named image for every authored game card.

The app does not currently serve this directory. Generated files become
runtime assets only after a deck-background integration explicitly references
them.
