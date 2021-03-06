#!/bin/sh
#
# Nanocloud Community, a comprehensive platform to turn any application
# into a cloud solution.
#
# Copyright (C) 2015 Nanocloud Software
#
# This file is part of Nanocloud community.
#
# Nanocloud community is free software; you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as
# published by the Free Software Foundation, either version 3 of the
# License, or (at your option) any later version.
#
# Nanocloud community is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.

clone() {
    local pkg="$1"
    local rev="$2"
    local url="$3"

    : ${url:=https://$pkg}
    local target="vendor/$pkg"

    echo -n "$pkg @ $rev: "

    if [ -d "$target" ]; then
        echo -n 'rm old, '
        rm -rf "$target"
    fi

    echo -n 'clone, '
    git clone --quiet --no-checkout "$url" "$target"
    ( cd "$target" && git checkout --quiet "$rev" && git reset --quiet --hard "$rev" )

    echo -n 'rm .git, '
    ( cd "$target" && rm -rf .git )

    echo -n 'rm vendor, '
    ( cd "$target" && rm -rf vendor Godeps/_workspace )

    echo done
}

clone github.com/labstack/echo b676ad11cf0d2c928012c0438df67a04b7c2c37f
clone github.com/lib/pq 165a3529e799da61ab10faed1fabff3662d6193f
clone github.com/Sirupsen/logrus 219c8cb75c258c552e999735be6df753ffc7afdc
clone github.com/satori/go.uuid e673fdd4dea8a7334adbbe7f57b7e4b00bdc5502
clone github.com/labstack/gommon c7a42f4800da9d39225ce15411f48288d622e517
clone github.com/mattn/go-colorable 9cbef7c35391cca05f15f8181dc0b18bc9736dbb
clone github.com/mattn/go-isatty 56b76bdf51f7708750eac80fa38b952bb9f32639
clone github.com/gedex/inflector 8c0e57904488c554ab26caec525db5c92b23f051
clone github.com/manyminds/api2go 0f0aba686fffe91897ee4b7e1d682efbfcbac7fd
clone github.com/julienschmidt/httprouter 77366a47451a56bb3ba682481eed85b64fea14e8
clone golang.org/x/crypto de93d05161db39bcbd84d3da2e54c4a18f37f0b1 https://go.googlesource.com/crypto
clone golang.org/x/net e7da8edaa52631091740908acaf2c2d4c9b3ce90 https://go.googlesource.com/net
clone gopkg.in/asn1-ber.v1 4e86f4367175e39f69d9358a5f17b4dda270378d https://gopkg.in/asn1-ber.v1
clone gopkg.in/ldap.v2 07a7330929b9ee80495c88a4439657d89c7dbd87 https://gopkg.in/ldap.v2
