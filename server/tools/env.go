package tools

import (
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_resolve"
)

// AppendExtraPaths appends extra paths to the PATH variable in the given
// environment slice and returns the modified slice.
// Delegates to tool_resolve.AppendExtraPaths.
func AppendExtraPaths(env []string) []string {
	return tool_resolve.AppendExtraPaths(env)
}
