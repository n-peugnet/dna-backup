/* Copyright (C) 2021 Nicolas Peugnet <n.peugnet@free.fr>

   This file is part of dna-backup.

   dna-backup is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   dna-backup is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with dna-backup.  If not, see <https://www.gnu.org/licenses/>. */

package utils

import (
	"fmt"
	"strings"
)

func Unprefix(path string, prefix string) (string, error) {
	if !strings.HasPrefix(path, prefix) {
		return path, fmt.Errorf("%q is not prefixed by %q", path, prefix)
	} else {
		return path[len(prefix):], nil
	}
}
