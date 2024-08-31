# ofc

Habbo Origins Figure Converter

## Archived
Now part of [nx](https://github.com/xabbo/nx) as [cmd/figure/convert](https://github.com/xabbo/nx/blob/22d25654661cc9b7fa6a7de8cfb293a75932e687/cmd/nx/cmd/figure/convert/convert.go):
```sh
$ nx figure convert 2951027534180012550415002
sh-295-1198.lg-275-1198.hd-180-1026.ch-255-1198.hr-150-1042.ha-1003-1042
```

### Usage

```sh
$ ./ofc 2951027534180012550415002
Loading modern figure data... ok
Loading origins figure data... ok
sh-295-1198.lg-275-1198.hd-180-1026.ch-255-1198.hr-150-1042.ha-1003-1042
```

### Build

```sh
$ go build -o bin/ && cd bin
```
