# dzip
`dzip` is small utility that implements subset of functionality of standard `zip` tool
that can be found on Unix systems, but ensures that for same input, output is always
the same if content of input files is not changed. This means that `md5` or `sha1` 
hash of output zip file should not change if inputs files have not changed. 

It is not drop-in replacement of `zip`, almost everything except basic zipping functionality
is missing (some functionalities might be added).

As opposed to `zip`, `dzip` handles directories automatically, without `-r` flag. 

## Install
Fetch latest version from [releases](https://github.com/delicb/dzip/releases) page.

Of, if you have go installed:
```shell
go install github.com/delicb/dzip@latest
```

Or, you know, clone and build:
```shell
git clone https://github.com/delicb/dzip.git && cd dzip && go install
```

## Usage
Invoke `dzip` with path to output file as first argument and list of input paths.

For example
```shell
dzip compressed.zip input1.txt input2
```

This will create `compressed.zip` file. 

## Why
Notice the following flow:
```shell
$ echo "content" > file
$ zip zipped.zip file
  adding: file (stored 0%)
$ sha1sum zipped.zip
deae7f2b7ca74c6456628007a6a5483821b64ac6  zipped.zip
$ touch file
$ zip zipped2.zip file
  adding: file (stored 0%)
$ sha1sum zipped2.zip
5522a73b8861f0daf56844835f6a981a605f18e4  zipped2.zip
```

Now, compare the same flow with `dzip`
```shell
$ echo "content" > file
$ dzip zipped.zip file
$ sha1sum zipped.zip
775a7ce2fb33ddcc4297897c39891907026a6e54  zipped.zip
$ touch file
$ dzip zipped2.zip file
$ sha1sum zipped2.zip
775a7ce2fb33ddcc4297897c39891907026a6e54  zipped2.zip
```

Even when file content is not changed, `zip` produces output with a different hash.

A lot of tools use checksum of input file to determine if further actions are needed. 
This is especially important in CI systems, since expensive operations can be prevented
if it is determined that input has not changed. Terraform, AWS CDK and similar use this 
approach. 

Standard `zip` tool changes its output checksum even if content compressed is not changed. 
This is because it is using meta information like:
- modification time
- permissions on a file
- file ordering is important

When used in CI system, modification time of a file is something that should not be
trusted. Even locally, if you change `git` branch, for example, modification time can
be change even if content is he same. Or if you run a build and get the same output as 
in previous build (content and checksum), modification time of output file will be 
different and `zip` will treat it differently.

Simple solution would be to use some other compression format (and I recommend that), but
some systems require zip (e.g. AWS Lambda), hence this tool.

## Caveats
Be warned that `dzip` changes some behaviors of normal `zip` util. 

- does not store modification time of files
- does not preserve permissions or any file attributes. 
  If file does not have exec permission (for user, group or other)
  it's permission will be stored as 0444. If it does have any
  exec permission, permissions 0555 will be stored. With this, 
  upon unzipping, executable files will still be executable, but
  other permissions will not be transferred, so keep that in mind.
- does not store any extra attributes
- no way to specify store-only, `dzip` always does compression
- potentially other things

## Further development
For now, this works for me. Except potential fixes, I do not plan to add more 
functionality until needed.

If I happen to need more functionality that exists in `zip` tool, I might add it.

However, if you find this tool useful and find some missing functionality, please let me
know by opening a ticket or sending pull request, as mentioned in Contribution section. 

## Contribution
Tool is created to scratch a personal itch, but if you find it useful and want to contribute,
please feel free to open an issue or, better, send a pull request. 

## Author
- Bojan Delic <bojan@delic.in.rs>
