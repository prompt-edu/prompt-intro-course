import { useMutation, useQueryClient } from '@tanstack/react-query'
import type { CoursePhaseParticipationWithStudent } from '@tumaet/prompt-shared-state'
import {
  Alert,
  AlertDescription,
  Badge,
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Command,
  CommandEmpty,
  CommandInput,
  CommandItem,
  CommandList,
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@tumaet/prompt-ui-components'
import {
  ArrowLeftRight,
  Loader2,
  Pencil,
  Plus,
  Save,
  Undo2,
  UserPlus,
  Users,
  X,
} from 'lucide-react'
import { useCallback, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import type { DeveloperProfile } from '../../../interfaces/DeveloperProfile'
import type { PeerAssignment } from '../../../interfaces/PeerAssignment'
import type { Seat } from '../../../interfaces/Seat'
import type { Tutor } from '../../../interfaces/Tutor'
import { updatePeerAssignments } from '../../../network/mutations/updatePeerAssignments'

interface PeerGroupListProps {
  peerAssignments: PeerAssignment[]
  seats: Seat[]
  tutors: Tutor[]
  developerProfiles: DeveloperProfile[]
  participations: CoursePhaseParticipationWithStudent[]
}

interface PeerGroup {
  members: string[] // courseParticipationIDs
}

/** Convert flat bidirectional PeerAssignment[] into connected-component PeerGroup[]. */
function buildGroups(assignments: PeerAssignment[]): PeerGroup[] {
  const adjacency = new Map<string, Set<string>>()
  for (const a of assignments) {
    if (!adjacency.has(a.studentID)) adjacency.set(a.studentID, new Set())
    adjacency.get(a.studentID)!.add(a.peerID)
  }

  const visited = new Set<string>()
  const groups: PeerGroup[] = []

  for (const studentId of adjacency.keys()) {
    if (visited.has(studentId)) continue
    const component: string[] = []
    const queue = [studentId]
    while (queue.length > 0) {
      const current = queue.shift()!
      if (visited.has(current)) continue
      visited.add(current)
      component.push(current)
      const neighbors = adjacency.get(current)
      if (neighbors) {
        for (const n of neighbors) {
          if (!visited.has(n)) queue.push(n)
        }
      }
    }
    groups.push({ members: component })
  }

  return groups
}

/** Convert PeerGroup[] back to flat bidirectional PeerAssignment[]. */
function groupsToAssignments(groups: PeerGroup[]): PeerAssignment[] {
  const assignments: PeerAssignment[] = []
  for (const group of groups) {
    for (let i = 0; i < group.members.length; i++) {
      for (let j = 0; j < group.members.length; j++) {
        if (i !== j) {
          assignments.push({ studentID: group.members[i], peerID: group.members[j] })
        }
      }
    }
  }
  return assignments
}

/** Group PeerGroup[] by tutor based on seat assignments. */
function groupByTutor(
  groups: PeerGroup[],
  studentSeatMap: Map<string, Seat>,
): Map<string, PeerGroup[]> {
  const byTutor = new Map<string, PeerGroup[]>()
  for (const group of groups) {
    const seat = studentSeatMap.get(group.members[0])
    const tutorId = seat?.assignedTutor ?? 'unassigned'
    if (!byTutor.has(tutorId)) byTutor.set(tutorId, [])
    byTutor.get(tutorId)!.push(group)
  }
  return byTutor
}

export const PeerGroupList = ({
  peerAssignments,
  seats,
  tutors,
  developerProfiles,
  participations,
}: PeerGroupListProps) => {
  const { phaseId } = useParams<{ phaseId: string }>()
  const queryClient = useQueryClient()
  const [isEditing, setIsEditing] = useState(false)
  const [editGroups, setEditGroups] = useState<PeerGroup[]>([])
  const [error, setError] = useState<string | null>(null)
  const [addingToGroup, setAddingToGroup] = useState<number | null>(null)

  // Build lookup maps
  const studentSeatMap = useMemo(() => {
    const map = new Map<string, Seat>()
    for (const seat of seats) {
      if (seat.assignedStudent) {
        map.set(seat.assignedStudent, seat)
      }
    }
    return map
  }, [seats])

  const profileMap = useMemo(() => {
    const map = new Map<string, DeveloperProfile>()
    for (const p of developerProfiles) {
      map.set(p.courseParticipationID, p)
    }
    return map
  }, [developerProfiles])

  const participationMap = useMemo(() => {
    const map = new Map<string, CoursePhaseParticipationWithStudent>()
    for (const p of participations) {
      if (p.courseParticipationID) {
        map.set(p.courseParticipationID, p)
      }
    }
    return map
  }, [participations])

  const tutorMap = useMemo(() => {
    const map = new Map<string, Tutor>()
    for (const t of tutors) {
      map.set(t.id, t)
    }
    return map
  }, [tutors])

  // Current groups from server data
  const serverGroups = useMemo(() => buildGroups(peerAssignments), [peerAssignments])

  // Active groups (edit mode uses local state, otherwise server data)
  const activeGroups = isEditing ? editGroups : serverGroups

  // Group by tutor for display
  const tutorPeerGroups = useMemo(
    () => groupByTutor(activeGroups, studentSeatMap),
    [activeGroups, studentSeatMap],
  )

  // All students with seat assignments (potential peer group members)
  const allAssignedStudents = useMemo(() => {
    return seats.filter((s) => s.assignedStudent && !s.isTutorSeat).map((s) => s.assignedStudent!)
  }, [seats])

  // Students not in any peer group (for adding)
  const unassignedStudents = useMemo(() => {
    const inGroup = new Set(activeGroups.flatMap((g) => g.members))
    return allAssignedStudents.filter((id) => !inGroup.has(id))
  }, [activeGroups, allAssignedStudents])

  const saveMutation = useMutation({
    mutationFn: () => updatePeerAssignments(phaseId ?? '', groupsToAssignments(editGroups)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['peerAssignments', phaseId] })
      setIsEditing(false)
      setError(null)
    },
    onError: () => setError('Failed to save peer assignments.'),
  })

  const getStudentLabel = useCallback(
    (studentId: string) => {
      const participation = participationMap.get(studentId)
      const profile = profileMap.get(studentId)
      const seat = studentSeatMap.get(studentId)

      const name = participation
        ? `${participation.student?.firstName ?? ''} ${participation.student?.lastName ?? ''}`.trim()
        : studentId.slice(0, 8)
      const gitlab = profile?.gitLabUsername ?? ''
      const seatName = seat?.seatName ?? ''

      return { name, gitlab, seatName }
    },
    [participationMap, profileMap, studentSeatMap],
  )

  const enterEditMode = useCallback(() => {
    setEditGroups(serverGroups.map((g) => ({ members: [...g.members] })))
    setIsEditing(true)
    setError(null)
  }, [serverGroups])

  const cancelEdit = useCallback(() => {
    setIsEditing(false)
    setEditGroups([])
    setError(null)
    setAddingToGroup(null)
  }, [])

  const removeFromGroup = useCallback((groupIndex: number, memberId: string) => {
    setEditGroups((prev) => {
      const updated = prev.map((g) => ({ members: [...g.members] }))
      updated[groupIndex].members = updated[groupIndex].members.filter((m) => m !== memberId)
      // Remove empty groups
      return updated.filter((g) => g.members.length >= 2)
    })
  }, [])

  const addToGroup = useCallback((groupIndex: number, studentId: string) => {
    setEditGroups((prev) => {
      const updated = prev.map((g) => ({ members: [...g.members] }))
      updated[groupIndex].members.push(studentId)
      return updated
    })
    setAddingToGroup(null)
  }, [])

  const createNewGroup = useCallback((studentId: string) => {
    setEditGroups((prev) => [...prev, { members: [studentId] }])
  }, [])

  // Flatten to find the actual flat index of a group across tutors
  const flatGroupIndex = useCallback(
    (tutorId: string, localIdx: number): number => {
      let offset = 0
      for (const [tid, groups] of tutorPeerGroups.entries()) {
        if (tid === tutorId) return offset + localIdx
        offset += groups.length
      }
      return -1
    },
    [tutorPeerGroups],
  )

  // Map from active group → index in editGroups array
  const getEditGroupIndex = useCallback(
    (group: PeerGroup): number => {
      return editGroups.findIndex(
        (g) =>
          g.members.length === group.members.length &&
          g.members.every((m) => group.members.includes(m)),
      )
    },
    [editGroups],
  )

  if (peerAssignments.length === 0 && !isEditing) {
    return (
      <Card>
        <CardContent className='text-center py-8'>
          <Users className='h-12 w-12 mx-auto text-muted-foreground mb-2' />
          <h3 className='text-lg font-medium mb-2'>No Peer Assignments</h3>
          <p className='text-muted-foreground max-w-md mx-auto'>
            Click &quot;Generate Groups&quot; to automatically create peer review groups within each
            tutor group.
          </p>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className='space-y-4'>
      {/* Edit controls */}
      <div className='flex items-center gap-2'>
        {!isEditing ? (
          <Button variant='outline' size='sm' onClick={enterEditMode}>
            <Pencil className='h-4 w-4 mr-1' />
            Edit Groups
          </Button>
        ) : (
          <>
            <Button
              size='sm'
              onClick={() => saveMutation.mutate()}
              disabled={saveMutation.isPending}
            >
              {saveMutation.isPending ? (
                <Loader2 className='h-4 w-4 animate-spin mr-1' />
              ) : (
                <Save className='h-4 w-4 mr-1' />
              )}
              Save Changes
            </Button>
            <Button
              variant='outline'
              size='sm'
              onClick={cancelEdit}
              disabled={saveMutation.isPending}
            >
              <Undo2 className='h-4 w-4 mr-1' />
              Cancel
            </Button>
          </>
        )}
      </div>

      {error && (
        <Alert variant='destructive'>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {/* Unassigned students (edit mode only) */}
      {isEditing && unassignedStudents.length > 0 && (
        <Card data-testid='peer-unassigned-students'>
          <CardHeader className='pb-3'>
            <div className='flex items-center justify-between'>
              <CardTitle className='text-base'>Unassigned Students</CardTitle>
              <Badge variant='destructive'>{unassignedStudents.length} unassigned</Badge>
            </div>
          </CardHeader>
          <CardContent>
            <div className='flex flex-wrap gap-2'>
              {unassignedStudents.map((studentId) => {
                const { name, seatName } = getStudentLabel(studentId)
                return (
                  <Popover key={studentId}>
                    <PopoverTrigger asChild>
                      <Button variant='outline' size='sm'>
                        <UserPlus className='h-3 w-3 mr-1' />
                        {name}
                        {seatName && (
                          <Badge variant='outline' className='ml-1 text-xs'>
                            {seatName}
                          </Badge>
                        )}
                      </Button>
                    </PopoverTrigger>
                    <PopoverContent className='w-64 p-0'>
                      <Command>
                        <CommandInput placeholder='Search group...' />
                        <CommandList>
                          <CommandEmpty>No groups found</CommandEmpty>
                          {editGroups.map((group, idx) => {
                            return (
                              <CommandItem
                                key={group.members.join('-')}
                                onSelect={() => addToGroup(idx, studentId)}
                              >
                                Group:{' '}
                                {group.members.map((m) => getStudentLabel(m).name).join(', ')}
                              </CommandItem>
                            )
                          })}
                          <CommandItem onSelect={() => createNewGroup(studentId)}>
                            <Plus className='h-3 w-3 mr-1' />
                            Start new group
                          </CommandItem>
                        </CommandList>
                      </Command>
                    </PopoverContent>
                  </Popover>
                )
              })}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Peer groups by tutor */}
      {Array.from(tutorPeerGroups.entries()).map(([tutorId, groups]) => {
        const tutor = tutorMap.get(tutorId)
        const tutorLabel = tutor
          ? `${tutor.firstName} ${tutor.lastName}`
          : tutorId === 'unassigned'
            ? 'Unassigned'
            : tutorId.slice(0, 8)
        const studentCount = groups.reduce((sum, g) => sum + g.members.length, 0)

        return (
          <Card key={tutorId} data-testid={`peer-tutor-card-${tutorId}`}>
            <CardHeader className='pb-3'>
              <div className='flex items-center justify-between'>
                <CardTitle className='text-base'>{tutorLabel}</CardTitle>
                <Badge variant='secondary'>{studentCount} students</Badge>
              </div>
            </CardHeader>
            <CardContent>
              <div className='space-y-2'>
                {groups.map((group) => {
                  const editIdx = isEditing ? getEditGroupIndex(group) : -1

                  return (
                    <div
                      key={group.members.join('-')}
                      className='flex items-center gap-2 p-2 rounded-md bg-muted/30 flex-wrap'
                      data-testid='peer-group'
                    >
                      {group.members.map((memberId, memberIdx) => {
                        const { name, gitlab, seatName } = getStudentLabel(memberId)
                        return (
                          <div
                            key={memberId}
                            className='flex items-center gap-2'
                            data-testid={`peer-group-member-${memberId}`}
                          >
                            {memberIdx > 0 && (
                              <ArrowLeftRight className='h-3 w-3 text-muted-foreground' />
                            )}
                            <div className='text-sm flex items-center gap-1'>
                              <span className='font-medium'>{name}</span>
                              {gitlab && <span className='text-muted-foreground'>({gitlab})</span>}
                              {seatName && (
                                <Badge variant='outline' className='text-xs'>
                                  {seatName}
                                </Badge>
                              )}
                              {isEditing && group.members.length > 1 && (
                                <Button
                                  variant='ghost'
                                  size='sm'
                                  className='h-5 w-5 p-0 text-destructive hover:text-destructive'
                                  onClick={() => removeFromGroup(editIdx, memberId)}
                                >
                                  <X className='h-3 w-3' />
                                </Button>
                              )}
                            </div>
                          </div>
                        )
                      })}

                      <div className='ml-auto flex items-center gap-1'>
                        {isEditing && unassignedStudents.length > 0 && (
                          <Popover
                            open={addingToGroup === editIdx}
                            onOpenChange={(open) => setAddingToGroup(open ? editIdx : null)}
                          >
                            <PopoverTrigger asChild>
                              <Button variant='ghost' size='sm' className='h-6 w-6 p-0'>
                                <Plus className='h-3 w-3' />
                              </Button>
                            </PopoverTrigger>
                            <PopoverContent className='w-64 p-0'>
                              <Command>
                                <CommandInput placeholder='Search student...' />
                                <CommandList>
                                  <CommandEmpty>No students found</CommandEmpty>
                                  {unassignedStudents.map((studentId) => {
                                    const { name, seatName } = getStudentLabel(studentId)
                                    return (
                                      <CommandItem
                                        key={studentId}
                                        onSelect={() => addToGroup(editIdx, studentId)}
                                      >
                                        {name}
                                        {seatName && ` (${seatName})`}
                                      </CommandItem>
                                    )
                                  })}
                                </CommandList>
                              </Command>
                            </PopoverContent>
                          </Popover>
                        )}
                        <Badge variant='secondary' className='text-xs'>
                          {group.members.length === 2
                            ? 'Pair'
                            : group.members.length === 3
                              ? 'Triple'
                              : group.members.length === 4
                                ? 'Quad'
                                : `${group.members.length}`}
                        </Badge>
                      </div>
                    </div>
                  )
                })}
              </div>
            </CardContent>
          </Card>
        )
      })}
    </div>
  )
}
