# term

`term` allows you to record your terminal without any external dependencies
on remote services or dynamic packages.  There have been many projects that
do this same thing but it's sometimes hard to get the software working.
`term` allows you to record to a file and play it back which it's located on
your local system or a remote url.

## welcome recording

To play the welcome recording just download the `term` cli tool and run:

```bash
term play https://raw.githubusercontent.com/crosbymichael/term/master/welcome.term
```


### record your term

```bash
term rec <filename>
# exit when you are finished 
```

### playback a localfile

```bash
term play <filename>
```

### playback a url

```bash
term play http://<someurl>
```
