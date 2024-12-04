// Copyright 2022 Evmos Foundation
// This file is part of the Evmos Network packages.
//
// Evmos is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Evmos packages are distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Evmos packages. If not, see https://github.com/evmos/evmos/blob/main/LICENSE

// These accounts represent the affected accounts during the Claims decay bug

// The decay occurred before planned and the corresponding claimed amounts
// were less than supposed to be

package v12

// Accounts holds the missing claim amount to the corresponding account
var Accounts = [5][2]string{
	{"guru10jmp6sgh4cc6zt3e8gw05wavvejgr5pwggsdaj", "11874559096980682752"},
	{"guru1cml96vmptgw99syqrrz8az79xer2pcgpawsch9", "2616281440833449984"},
	{"guru1jcltmuhplrdcwp7stlr4hlhlhgd4htqhtx0smk", "291140584219587084288"},
	{"guru1gzsvk8rruqn2sx64acfsskrwy8hvrmaf6dvhj3", "1648928122612736"},
	{"guru1fx944mzagwdhx0wz7k9tfztc8g3lkfk6ecee3f", "2876440839872856064"},
}
