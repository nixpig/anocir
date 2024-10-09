package filesystem

import "golang.org/x/sys/unix"

var mountOptions = map[string]struct {
	No   bool
	Flag uintptr
}{
	"async":         {No: false, Flag: unix.MS_SYNCHRONOUS},
	"atime":         {No: false, Flag: unix.MS_NOATIME},
	"bind":          {No: false, Flag: unix.MS_BIND},
	"defaults":      {No: false, Flag: 0},
	"dev":           {No: false, Flag: unix.MS_NODEV},
	"diratime":      {No: false, Flag: unix.MS_NODIRATIME},
	"dirsync":       {No: false, Flag: unix.MS_DIRSYNC},
	"exec":          {No: false, Flag: unix.MS_NOEXEC},
	"iversion":      {No: false, Flag: unix.MS_I_VERSION},
	"lazytime":      {No: false, Flag: unix.MS_LAZYTIME},
	"loud":          {No: false, Flag: unix.MS_SILENT},
	"noatime":       {No: true, Flag: unix.MS_NOATIME},
	"nodev":         {No: true, Flag: unix.MS_NODEV},
	"nodiratime":    {No: true, Flag: unix.MS_NODIRATIME},
	"noexec":        {No: true, Flag: unix.MS_NOEXEC},
	"noiversion":    {No: true, Flag: unix.MS_I_VERSION},
	"nolazytime":    {No: true, Flag: unix.MS_LAZYTIME},
	"norelatime":    {No: true, Flag: unix.MS_RELATIME},
	"nostrictatime": {No: true, Flag: unix.MS_STRICTATIME},
	"nosuid":        {No: true, Flag: unix.MS_NOSUID},
	"nosymfollow":   {No: true, Flag: unix.MS_NOSYMFOLLOW},
	"private":       {No: false, Flag: unix.MS_PRIVATE},
	"rbind":         {No: false, Flag: unix.MS_BIND | unix.MS_REC},
	"relatime":      {No: false, Flag: unix.MS_RELATIME},
	"remount":       {No: false, Flag: unix.MS_REMOUNT},
	"ro":            {No: false, Flag: unix.MS_RDONLY},
	"rprivate":      {No: false, Flag: unix.MS_PRIVATE | unix.MS_REC},
	"rshared":       {No: false, Flag: unix.MS_SHARED | unix.MS_REC},
	"rslave":        {No: false, Flag: unix.MS_SLAVE | unix.MS_REC},
	"runbindable":   {No: false, Flag: unix.MS_UNBINDABLE | unix.MS_REC},
	"rw":            {No: false, Flag: unix.MS_RDONLY},
	"shared":        {No: false, Flag: unix.MS_SHARED},
	"silent":        {No: false, Flag: unix.MS_SILENT},
	"slave":         {No: false, Flag: unix.MS_SLAVE},
	"strictatime":   {No: false, Flag: unix.MS_STRICTATIME},
	"suid":          {No: false, Flag: unix.MS_NOSUID},
	"sync":          {No: false, Flag: unix.MS_SYNCHRONOUS},
	"unbindable":    {No: false, Flag: unix.MS_UNBINDABLE},
}
