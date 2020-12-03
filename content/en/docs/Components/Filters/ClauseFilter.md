---
title: "ClauseFilter"
weight: 8
date: 2020-12-03
---
## Filter *ClauseFilter*

### Overview

Discard records which do not match a clause given as a boolean S-expression.  
 Check the filter documentation for some examples.  


### ClauseFilter boolean expression format

This document describes the s-expression format used in ClauseFilter.  


The format uses s-expressions.  
 Empty string matches anything (i.  
e.  
 all records
will pass the expression).  


There are only three keywords: and, or, not

If an s-expression starts with any other name, it is assumed to be the name of
a field and it should be paired with the desired value to match against.  


    Must match both X and Y to pass:
    (and X Y)

    You can use more than 2 arguments:
    (and X Y Z A B C)

    Must match either X or Y to pass:
    (or X Y)

    Must NOT match X to pass:
    (not X)

    Field must equal value to pass:
    (FIELD VALUE)

    e.  
g.  

    (fieldName somevalue)

    Matches anything (because only one argument)
    (and X)

    Matches nothing
    (and)

    Matches anything
    (or)

Examples:

    (and (fieldName value1) (anotherFieldName value2))

    (or (fieldName value1) (fieldName value2))

	(not (or (fieldName value1) (fieldName value2)))

    (or
      (and (fieldName value1)
           (anotherFieldName value3))
      (and (fieldName value2)
           (anotherFieldName value4)))


### Configuration

Keys available in the `[filter.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| Clause| string| ""| false| Boolean formula describing which events to let through. If empty, let everything through.|

