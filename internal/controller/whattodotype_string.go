// Code generated by "stringer -type=whatToDoType"; DO NOT EDIT.

package controller

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[shouldCreatePreMigrationJob-17]
	_ = x[shouldCreatePostMigrationJob-18]
	_ = x[shouldSetMigratingStatus-19]
	_ = x[shouldUnsetMigratingStatus-20]
	_ = x[shouldCreateOrUpdateDeploysWithSpec-21]
	_ = x[shouldCreateOrUpdateDeploysWithMigratingImages-22]
	_ = x[shouldDeletePostMigrationJob-23]
	_ = x[shouldDeletePreMigrationJob-24]
	_ = x[shouldDoNothing-25]
}

const _whatToDoType_name = "shouldCreatePreMigrationJobshouldCreatePostMigrationJobshouldSetMigratingStatusshouldUnsetMigratingStatusshouldCreateOrUpdateDeploysWithSpecshouldCreateOrUpdateDeploysWithMigratingImagesshouldDeletePostMigrationJobshouldDeletePreMigrationJobshouldDoNothing"

var _whatToDoType_index = [...]uint16{0, 27, 55, 79, 105, 140, 186, 214, 241, 256}

func (i whatToDoType) String() string {
	i -= 17
	if i < 0 || i >= whatToDoType(len(_whatToDoType_index)-1) {
		return "whatToDoType(" + strconv.FormatInt(int64(i+17), 10) + ")"
	}
	return _whatToDoType_name[_whatToDoType_index[i]:_whatToDoType_index[i+1]]
}
