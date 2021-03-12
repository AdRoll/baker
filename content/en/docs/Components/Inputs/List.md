---
title: "List"
weight: 4
date: 2021-03-12
---
{{% pageinfo color="primary" %}}

**Read the [API documentation &raquo;](https://pkg.go.dev/github.com/AdRoll/baker/input#List)**
{{% /pageinfo %}}

## Input *List*

### Overview
This input fetches logs from a predefined list of local or remote sources. The "Files"
configuration variable is a list of "file specifiers". Each "file specifier" can be:

  * A local file path on the filesystem: the log file at that path will be processed
  * A HTTP/HTTPS URL: the log file at that URL will be downloaded and processed
  * A S3 URL: the log file at that URL that will be downloaded and processed
  * "@" followed by a local path pointing to a file: the file is expected to be a text file
    and each line will be read and parsed as a "file specifier"
  * "@" followed by a HTTP/HTTPS URL: the text file pointed by the URL will be downloaded,
    and each line will be read and parsed as a "file specifier"
  * "@" followed by a S3 URL pointing to a file: the text file pointed by the URL will be
    downloaded, and each line will be read and parsed as a "file specifier"
  * "@" followed by a local path pointing to a directory (must end with a slash): the directory will be recursively
    walked, and all files matching the "MatchPath" option regexp will be processed as logfiles
  * "@" followed by a S3 URL pointing to a directory: the directory on S3 will be recursively
    walked, and all files matching the "MatchPath" option regexp will be processed as logfiles
  * "-": the contents of a log file will be read from stdin and processed
  * "@-": each line read from stdin will be parsed as a "file specifier"

All records produced by this input contain 2 metadata values:
  * url: the files that originally contained the record
  * last_modified: the last modification datetime of the above file


### Configuration

Keys available in the `[input.config]` section:

|Name|Type|Default|Required|Description|
|----|:--:|:-----:|:------:|-----------|
| Files| array of strings| ["-"]| false| List of log-files, directories and/or list-files to process|
| MatchPath| string| ".*\.log\.gz"| false| regexp to filter files in specified directories|
| Region| string| "us-west-2"| false| AWS Region for fetching from S3|

