// Package graph implements unicode commit graph rendering.
// This is a Go port of Neogit's graph/unicode.lua, itself a modified version
// of the vim-flog algorithm (https://github.com/rbong/vim-flog).
package graph

import (
	"fmt"
	"strings"
)

// Cell represents a single character in the graph with its color.
type Cell struct {
	Text  string // A single Unicode character (e.g., "│", "•", "─")
	Color string // Color name (e.g., "Purple")
	OID   string // Commit hash for commit rows, empty for connector rows
}

// Row is a single row of graph cells.
type Row = []Cell

// CommitInput is the minimal commit data needed for graph building.
type CommitInput struct {
	OID     string   // Full commit hash
	Parents []string // Parent commit hashes
}

// Unicode box-drawing characters matching Neogit exactly.
const (
	currentCommitStr    = "• "
	commitBranchStr     = "│ "
	commitEmptyStr      = "  "
	complexMergeStr1    = "┬┊"
	complexMergeStr2    = "╰┤"
	mergeAllStr         = "┼"
	mergeJumpStr        = "┊"
	mergeUpDownLeftStr  = "┤"
	mergeUpDownRightStr = "├"
	mergeUpDownStr      = "│"
	mergeUpLeftRightStr = "┴"
	mergeUpLeftStr      = "╯"
	mergeUpRightStr     = "╰"
	mergeUpStr          = " "
	mergeDownLeftRight  = "┬"
	mergeDownLeftStr    = "╮"
	mergeDownRightStr   = "╭"
	mergeLeftRightStr   = "─"
	mergeEmptyStr       = " "
	missingParentStr    = "┊ "
	missingParentBranch = "│ "
	missingParentEmpty  = "  "
)

// rawLine is an intermediate output line before splitting into cells.
type rawLine struct {
	text  string
	color string
	oid   string
}

// Build computes the unicode graph for a list of commits.
// Returns one or more Rows per commit: the first row has OID set (commit row
// with "•"), subsequent rows are connector rows with empty OID.
func Build(commits []CommitInput) ([]Row, error) {
	if len(commits) == 0 {
		return nil, nil
	}

	// Build set of all commit hashes for missing parent detection
	commitHashes := make(map[string]bool, len(commits))
	for _, c := range commits {
		if c.OID != "" {
			commitHashes[c.OID] = true
		}
	}

	var out []rawLine

	// Graph state (1-based indexing to match Lua algorithm)
	branchHashes := map[int]string{}  // column index -> hash
	branchIndexes := map[string]int{} // hash -> column index
	nbranches := 0

	for _, commit := range commits {
		if commit.OID == "" {
			continue
		}

		commitHash := commit.OID
		parents := commit.Parents
		// Filter out empty parent strings
		if len(parents) == 1 && parents[0] == "" {
			parents = nil
		}
		nparents := len(parents)

		parentHashSet := make(map[string]bool, nparents)
		for _, p := range parents {
			parentHashSet[p] = true
		}

		// Init commit output — use maps for sparse indexing like Lua tables
		commitPrefix := map[int]string{}
		ncommitStrings := 0
		mergeLine := map[int]string{}
		complexMergeLine := map[int]string{}
		nmergeStrings := 0
		missingParentsLine1 := map[int]string{}
		missingParentsLine2 := map[int]string{}
		nmissingParentsStrings := 0

		// Init visual data
		ncomplex := 0
		nmissingParents := 0

		// Init graph data
		nmergesLeft := 0
		nmergesRight := nparents + 1
		commitBranchIndex, commitHasBranch := branchIndexes[commitHash]
		movedParentBranchIndex := 0
		hasMovedParent := false
		ncommitBranches := nbranches
		if !commitHasBranch {
			ncommitBranches++
		}

		// Init indexes
		branchIndex := 1
		parentIndex := 1

		// Find the first empty (unassigned) parent
		for parentIndex <= nparents {
			if _, assigned := branchIndexes[parents[parentIndex-1]]; !assigned {
				break
			}
			parentIndex++
		}

		// Traverse old and new branches
		for branchIndex <= nbranches || nmergesRight > 0 {
			// Get branch data
			branchHash, hasBranchHash := branchHashes[branchIndex]
			isCommit := commitHasBranch && branchIndex == commitBranchIndex

			// Set merge info before updates
			mergeUp := hasBranchHash || (hasMovedParent && movedParentBranchIndex == branchIndex)
			mergeLeft := nmergesLeft > 0 && nmergesRight > 0
			isComplex := false
			isMissingParent := false

			// Handle commit
			if !hasBranchHash && !commitHasBranch {
				// Found empty branch and commit does not have a branch
				commitBranchIndex = branchIndex
				commitHasBranch = true
				isCommit = true
			}

			if isCommit {
				// Count commit merge
				nmergesRight--
				nmergesLeft++

				if hasBranchHash {
					// End of branch — remove branch
					delete(branchHashes, commitBranchIndex)
					delete(branchIndexes, commitHash)

					// Trim trailing empty branches
					for nbranches > 0 {
						if _, ok := branchHashes[nbranches]; ok {
							break
						}
						nbranches--
					}

					hasBranchHash = false
					branchHash = ""
				}

				if parentIndex > nparents && nmergesRight == 1 {
					// There is only one remaining parent, to the right — move it under the commit
					parentIndex = nparents
					for parentIndex >= 1 {
						pi, ok := branchIndexes[parents[parentIndex-1]]
						if !ok {
							pi = -1
						}
						if pi >= branchIndex {
							break
						}
						parentIndex--
					}

					parentHash := parents[parentIndex-1]
					parentBrIdx := branchIndexes[parentHash]

					// Remove old parent branch
					delete(branchHashes, parentBrIdx)
					delete(branchIndexes, parentHash)

					// Trim trailing empty branches
					for nbranches > 0 {
						if _, ok := branchHashes[nbranches]; ok {
							break
						}
						nbranches--
					}

					movedParentBranchIndex = parentBrIdx
					hasMovedParent = true

					nmergesRight++
				}
			}

			// Handle parents
			if !hasBranchHash && branchHash == "" && parentIndex <= nparents {
				// New parent
				parentHash := parents[parentIndex-1]

				// Set branch to parent
				branchIndexes[parentHash] = branchIndex
				branchHashes[branchIndex] = parentHash

				hasBranchHash = true
				branchHash = parentHash

				if branchIndex > nbranches {
					nbranches = branchIndex
				}

				// Jump to next available parent
				parentIndex++
				for parentIndex <= nparents {
					if _, assigned := branchIndexes[parents[parentIndex-1]]; !assigned {
						break
					}
					parentIndex++
				}

				// Count new parent merge
				nmergesRight--
				nmergesLeft++

				// Determine if parent is missing
				if hasBranchHash && !commitHashes[parentHash] {
					isMissingParent = true
					nmissingParents++
				}
			} else if (hasMovedParent && branchIndex == movedParentBranchIndex) ||
				(nmergesRight > 0 && parentHashSet[branchHash]) {
				// Existing parents

				// Count existing parent merge
				nmergesRight--
				nmergesLeft++

				// Determine if parent has a complex merge
				isComplex = mergeLeft && nmergesRight > 0
				if isComplex {
					ncomplex++
				}

				// Determine if parent is missing
				if hasBranchHash && !commitHashes[branchHash] {
					isMissingParent = true
					nmissingParents++
				}
			}

			// Draw commit lines
			if branchIndex <= ncommitBranches {
				ncommitStrings++

				if isCommit {
					commitPrefix[ncommitStrings] = currentCommitStr
				} else if mergeUp {
					commitPrefix[ncommitStrings] = commitBranchStr
				} else {
					commitPrefix[ncommitStrings] = commitEmptyStr
				}
			}

			// Update merge visual info
			nmergeStrings++

			// Draw merge lines
			if isComplex {
				mergeLine[nmergeStrings] = complexMergeStr1
				complexMergeLine[nmergeStrings] = complexMergeStr2
			} else {
				// Update merge info after drawing commit
				mergeUp = mergeUp || isCommit || (hasMovedParent && branchIndex == movedParentBranchIndex)
				mergeRight := nmergesLeft > 0 && nmergesRight > 0

				// Draw left character
				if branchIndex > 1 {
					if mergeLeft {
						mergeLine[nmergeStrings] = mergeLeftRightStr
					} else {
						mergeLine[nmergeStrings] = mergeEmptyStr
					}
					complexMergeLine[nmergeStrings] = mergeEmptyStr

					nmergeStrings++
				}

				// Draw right character
				if mergeUp {
					if hasBranchHash {
						if mergeLeft {
							if mergeRight {
								if isCommit {
									mergeLine[nmergeStrings] = mergeAllStr
								} else {
									mergeLine[nmergeStrings] = mergeJumpStr
								}
							} else {
								mergeLine[nmergeStrings] = mergeUpDownLeftStr
							}
						} else {
							if mergeRight {
								mergeLine[nmergeStrings] = mergeUpDownRightStr
							} else {
								mergeLine[nmergeStrings] = mergeUpDownStr
							}
						}
					} else {
						if mergeLeft {
							if mergeRight {
								mergeLine[nmergeStrings] = mergeUpLeftRightStr
							} else {
								mergeLine[nmergeStrings] = mergeUpLeftStr
							}
						} else {
							if mergeRight {
								mergeLine[nmergeStrings] = mergeUpRightStr
							} else {
								mergeLine[nmergeStrings] = mergeUpStr
							}
						}
					}
				} else {
					if hasBranchHash {
						if mergeLeft {
							if mergeRight {
								mergeLine[nmergeStrings] = mergeDownLeftRight
							} else {
								mergeLine[nmergeStrings] = mergeDownLeftStr
							}
						} else {
							if mergeRight {
								mergeLine[nmergeStrings] = mergeDownRightStr
							} else {
								return nil, fmt.Errorf("graph: internal error drawing graph: branch hash with no merge direction")
							}
						}
					} else {
						if mergeLeft {
							if mergeRight {
								mergeLine[nmergeStrings] = mergeLeftRightStr
							} else {
								return nil, fmt.Errorf("graph: internal error drawing graph: merge left without right")
							}
						} else {
							if mergeRight {
								return nil, fmt.Errorf("graph: internal error drawing graph: merge right without left")
							}
							mergeLine[nmergeStrings] = mergeEmptyStr
						}
					}
				}

				// Draw complex right char
				if hasBranchHash {
					complexMergeLine[nmergeStrings] = mergeUpDownStr
				} else {
					complexMergeLine[nmergeStrings] = mergeEmptyStr
				}
			}

			// Update visual missing parents info
			nmissingParentsStrings++

			// Draw missing parents lines
			if isMissingParent {
				missingParentsLine1[nmissingParentsStrings] = missingParentStr
				missingParentsLine2[nmissingParentsStrings] = missingParentEmpty
			} else if hasBranchHash {
				missingParentsLine1[nmissingParentsStrings] = missingParentBranch
				missingParentsLine2[nmissingParentsStrings] = missingParentBranch
			} else {
				missingParentsLine1[nmissingParentsStrings] = missingParentEmpty
				missingParentsLine2[nmissingParentsStrings] = missingParentEmpty
			}

			// Remove missing parent
			if isMissingParent && (!hasMovedParent || branchIndex != movedParentBranchIndex) {
				delete(branchHashes, branchIndex)
				delete(branchIndexes, branchHash)

				// Trim trailing empty branches
				for nbranches > 0 {
					if _, ok := branchHashes[nbranches]; ok {
						break
					}
					nbranches--
				}
			}

			branchIndex++
		}

		// Output — calculate whether certain lines should be outputted
		shouldOutMerge := nparents > 1 ||
			hasMovedParent ||
			(nparents == 0 && nbranches == 0)
		if !shouldOutMerge && nparents == 1 {
			if idx, ok := branchIndexes[parents[0]]; ok {
				shouldOutMerge = idx != commitBranchIndex
			} else {
				shouldOutMerge = true
			}
		}
		shouldOutComplex := shouldOutMerge && ncomplex > 0
		shouldOutMissingParents := nmissingParents > 0

		// Commit prefix row
		out = append(out, rawLine{
			text:  concatSparse(commitPrefix, ncommitStrings),
			color: "Purple",
			oid:   commitHash,
		})

		// Merge lines
		if shouldOutMerge {
			out = append(out, rawLine{
				text:  concatSparse(mergeLine, nmergeStrings),
				color: "Purple",
			})

			if shouldOutComplex {
				out = append(out, rawLine{
					text:  concatSparse(complexMergeLine, nmergeStrings),
					color: "Purple",
				})
			}
		}

		// Missing parents lines
		if shouldOutMissingParents {
			out = append(out, rawLine{
				text:  concatSparse(missingParentsLine1, nmissingParentsStrings),
				color: "Purple",
			})
			out = append(out, rawLine{
				text:  concatSparse(missingParentsLine2, nmissingParentsStrings),
				color: "Purple",
			})
		}
	}

	// Split each line into individual rune cells
	var rows []Row
	for _, line := range out {
		var row Row
		for _, r := range line.text {
			row = append(row, Cell{
				Text:  string(r),
				Color: line.color,
				OID:   line.oid,
			})
		}
		rows = append(rows, row)
	}

	return rows, nil
}

// concatSparse concatenates a sparse map of strings indexed 1..max in order.
func concatSparse(m map[int]string, max int) string {
	var b strings.Builder
	for i := 1; i <= max; i++ {
		b.WriteString(m[i]) // missing keys return ""
	}
	return b.String()
}
