package filesystem

import (
	"syscall"
)

var mountOptions = map[string]struct {
	No   bool
	Flag uintptr
}{
	"async":    {No: false, Flag: syscall.MS_SYNCHRONOUS},
	"atime":    {No: false, Flag: syscall.MS_NOATIME},
	"bind":     {No: false, Flag: syscall.MS_BIND},
	"defaults": {No: false, Flag: 0},
	"dev":      {No: false, Flag: syscall.MS_NODEV},
	"diratime": {No: false, Flag: syscall.MS_NODIRATIME},
	"dirsync":  {No: false, Flag: syscall.MS_DIRSYNC},
	"exec":     {No: false, Flag: syscall.MS_NOEXEC},
	"iversion": {No: false, Flag: syscall.MS_I_VERSION},
	// "lazytime":      {No: false, Flag: syscall.MS_LAZYTIME},
	"loud":       {No: false, Flag: syscall.MS_SILENT},
	"noatime":    {No: true, Flag: syscall.MS_NOATIME},
	"nodev":      {No: true, Flag: syscall.MS_NODEV},
	"nodiratime": {No: true, Flag: syscall.MS_NODIRATIME},
	"noexec":     {No: true, Flag: syscall.MS_NOEXEC},
	"noiversion": {No: true, Flag: syscall.MS_I_VERSION},
	// "nolazytime":    {No: true, Flag: syscall.MS_LAZYTIME},
	"norelatime":    {No: true, Flag: syscall.MS_RELATIME},
	"nostrictatime": {No: true, Flag: syscall.MS_STRICTATIME},
	"nosuid":        {No: true, Flag: syscall.MS_NOSUID},
	// "nosymfollow":   {No: true, Flag: syscall.MS_NOSYMFOLLOW},
	"private":     {No: false, Flag: syscall.MS_PRIVATE},
	"rbind":       {No: false, Flag: syscall.MS_BIND | syscall.MS_REC},
	"relatime":    {No: false, Flag: syscall.MS_RELATIME},
	"remount":     {No: false, Flag: syscall.MS_REMOUNT},
	"ro":          {No: false, Flag: syscall.MS_RDONLY},
	"rprivate":    {No: false, Flag: syscall.MS_PRIVATE | syscall.MS_REC},
	"rshared":     {No: false, Flag: syscall.MS_SHARED | syscall.MS_REC},
	"rslave":      {No: false, Flag: syscall.MS_SLAVE | syscall.MS_REC},
	"runbindable": {No: false, Flag: syscall.MS_UNBINDABLE | syscall.MS_REC},
	"rw":          {No: false, Flag: syscall.MS_RDONLY},
	"shared":      {No: false, Flag: syscall.MS_SHARED},
	"silent":      {No: false, Flag: syscall.MS_SILENT},
	"slave":       {No: false, Flag: syscall.MS_SLAVE},
	"strictatime": {No: false, Flag: syscall.MS_STRICTATIME},
	"suid":        {No: false, Flag: syscall.MS_NOSUID},
	"sync":        {No: false, Flag: syscall.MS_SYNCHRONOUS},
	"unbindable":  {No: false, Flag: syscall.MS_UNBINDABLE},
}
