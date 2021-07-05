# Go Farmer!!


Simple fyne app to register farms on threefold grid


## identity registration

![identity register](./img/registeridentity.png)

It's required to register an identity on the tfgrid explorer
requires:
- 3bot Name: alphanumeric ending with `.3bot`
- email
- words (mnemonics): if left empty it will get generated

## farm registration
![farm register](./img/registerfarm.png)

for the farm all you need is 
- farm name: alphanumeric
- tft wallet address
## running

- clone `https://github.com/xmonader/gofarmer`
- `go build`
- run using `./gofarmer`

### building 
- please note on ubuntu you may need to execute `sudo apt install libxxf86vm-dev`

## notes

seed file should exist or gets generated in `~/.config/tffarmer.seed`

the file looks like this:
```
"1.1.0"{"mnemonic":"some words","threebotid":2201}%   
```


