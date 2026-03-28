import { useMemo } from 'react'
import { ArrowLeftRight, Users } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle, Badge } from '@tumaet/prompt-ui-components'
import { PeerAssignment } from '../../../interfaces/PeerAssignment'
import { Seat } from '../../../interfaces/Seat'
import { Tutor } from '../../../interfaces/Tutor'
import { DeveloperProfile } from '../../../interfaces/DeveloperProfile'
import { CoursePhaseParticipationWithStudent } from '@tumaet/prompt-shared-state'

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

export const PeerGroupList = ({
  peerAssignments,
  seats,
  tutors,
  developerProfiles,
  participations,
}: PeerGroupListProps) => {
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

  // Group peer assignments into undirected peer groups per tutor
  const tutorPeerGroups = useMemo(() => {
    // Build adjacency: for each student, collect all their peers
    const adjacency = new Map<string, Set<string>>()
    for (const a of peerAssignments) {
      if (!adjacency.has(a.studentID)) adjacency.set(a.studentID, new Set())
      adjacency.get(a.studentID)!.add(a.peerID)
    }

    // Find connected components (pairs/triples)
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

    // Group by tutor
    const byTutor = new Map<string, PeerGroup[]>()
    for (const group of groups) {
      // Use the first member's seat to determine tutor
      const seat = studentSeatMap.get(group.members[0])
      const tutorId = seat?.assignedTutor ?? 'unassigned'
      if (!byTutor.has(tutorId)) byTutor.set(tutorId, [])
      byTutor.get(tutorId)!.push(group)
    }

    return byTutor
  }, [peerAssignments, studentSeatMap])

  const getStudentLabel = (studentId: string) => {
    const participation = participationMap.get(studentId)
    const profile = profileMap.get(studentId)
    const seat = studentSeatMap.get(studentId)

    const name = participation
      ? `${participation.student?.firstName ?? ''} ${participation.student?.lastName ?? ''}`.trim()
      : studentId.slice(0, 8)
    const gitlab = profile?.gitLabUsername ?? ''
    const seatName = seat?.seatName ?? ''

    return { name, gitlab, seatName }
  }

  if (peerAssignments.length === 0) {
    return (
      <div className='text-center py-8 bg-muted/30 rounded-lg'>
        <Users className='h-12 w-12 mx-auto text-muted-foreground mb-2' />
        <h3 className='text-lg font-medium mb-2'>No Peer Assignments</h3>
        <p className='text-muted-foreground max-w-md mx-auto'>
          Click &quot;Generate Groups&quot; to automatically create peer review groups within each
          tutor group.
        </p>
      </div>
    )
  }

  return (
    <div className='space-y-4'>
      {Array.from(tutorPeerGroups.entries()).map(([tutorId, groups]) => {
        const tutor = tutorMap.get(tutorId)
        const tutorLabel = tutor
          ? `${tutor.firstName} ${tutor.lastName}`
          : tutorId === 'unassigned'
            ? 'Unassigned'
            : tutorId.slice(0, 8)
        const studentCount = groups.reduce((sum, g) => sum + g.members.length, 0)

        return (
          <Card key={tutorId}>
            <CardHeader className='pb-3'>
              <div className='flex items-center justify-between'>
                <CardTitle className='text-base'>
                  {tutorLabel}
                </CardTitle>
                <Badge variant='secondary'>{studentCount} students</Badge>
              </div>
            </CardHeader>
            <CardContent>
              <div className='space-y-2'>
                {groups.map((group, groupIdx) => (
                  <div
                    key={groupIdx}
                    className='flex items-center gap-2 p-2 rounded-md bg-muted/30 flex-wrap'
                  >
                    {group.members.map((memberId, memberIdx) => {
                      const { name, gitlab, seatName } = getStudentLabel(memberId)
                      return (
                        <div key={memberId} className='flex items-center gap-2'>
                          {memberIdx > 0 && (
                            <ArrowLeftRight className='h-3 w-3 text-muted-foreground' />
                          )}
                          <div className='text-sm'>
                            <span className='font-medium'>{name}</span>
                            {gitlab && (
                              <span className='text-muted-foreground ml-1'>({gitlab})</span>
                            )}
                            {seatName && (
                              <Badge variant='outline' className='ml-1 text-xs'>
                                {seatName}
                              </Badge>
                            )}
                          </div>
                        </div>
                      )
                    })}
                    <Badge variant='secondary' className='text-xs ml-auto'>
                      {group.members.length === 2 ? 'Pair' : group.members.length === 3 ? 'Triple' : 'Quad'}
                    </Badge>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        )
      })}
    </div>
  )
}
