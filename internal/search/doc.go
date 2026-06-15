/*
Package search implements the core.Searcher port using either pure-Go recursive file
scanning or shelling out to the system 'rg' (ripgrep) tool as a fast path.

It filters out internal configuration and binding folders (like context/ and sessions/)
and parses files to return search results grouped by Dossiers and Artifacts.
*/
package search
