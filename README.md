

# EXPERIMENTAL

The code in this repository has an unstable API at the moment.

# Intro

This is a selection of interesting algorithms developed over the years, to work with
graphs and distances.

# strdist

An efficient implementation of the classic [Levenshtein distance](https://en.wikipedia.org/wiki/Levenshtein_distance) algorithm.

The algorithm was generalized to work over arbitrary costs, which is then leveraged as a fast
implementaion for wildcard matching (*, **, ?).

# listdist

This is strdist reshaped to work with lists instead of strings.

# tarjan

An implementation of [Tarjan's strongly connected components](http://en.wikipedia.org/wiki/Tarjan%27s_strongly_connected_components_algorithm) algorithm, which is often used as a
more resilient topological sort and cycle detector.

# assign

This is an implementation of the [Hungarian algorithm](https://en.wikipedia.org/wiki/Hungarian_algorithm) to solve the [assignment problem](https://en.wikipedia.org/wiki/Assignment_problem).

The algorithm was generalized to work with arbitrary cost types, to facilitate handling
of more involved relationships between objects.

