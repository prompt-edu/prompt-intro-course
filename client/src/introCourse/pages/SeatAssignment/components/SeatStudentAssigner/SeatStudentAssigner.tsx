import { useState, useMemo, useCallback } from 'react'
import {
  ChevronDown,
  ChevronUp,
  Download,
  AlertCircle,
  UserCheck,
  Users,
  Laptop,
  Sparkles,
} from 'lucide-react'
import { Seat } from '../../../../interfaces/Seat'
import { DeveloperWithProfile } from '../../interfaces/DeveloperWithProfile'
import { Tutor } from '../../../../interfaces/Tutor'
import { PeerAssignment } from '../../../../interfaces/PeerAssignment'
import { useUpdateSeats } from '../../hooks/useUpdateSeats'
import { useAssignStudents } from '../../hooks/useAssignStudents'
import { useDownloadAssignment } from '../../hooks/useDownloadAssignment'
import { smartAssign } from '../../utils/smartAssignment'
import { ResetSeatAssignmentDialog } from './ResetSeatAssignmentDialog'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
  Button,
  Alert,
  AlertDescription,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Badge,
} from '@tumaet/prompt-ui-components'

interface SeatStudentAssignerProps {
  seats: Seat[]
  developerWithProfiles: DeveloperWithProfile[]
  tutors: Tutor[]
  peerAssignments?: PeerAssignment[]
}

export const SeatStudentAssigner = ({
  seats,
  developerWithProfiles,
  tutors,
  peerAssignments,
}: SeatStudentAssignerProps) => {
  const [error, setError] = useState<string | null>(null)
  const [isCollapsed, setIsCollapsed] = useState(false)
  const totalStudents = developerWithProfiles.length
  const assignedStudents = seats.filter((seat) => seat.assignedStudent).length

  const assignmentStatus = useMemo<'none' | 'partial' | 'complete'>(() => {
    if (assignedStudents === 0) return 'none'
    if (assignedStudents < totalStudents) return 'partial'
    return 'complete'
  }, [assignedStudents, totalStudents])

  const mutation = useUpdateSeats(setError)

  // Assign function
  const assignStudents = useAssignStudents(seats, developerWithProfiles, setError)

  // Reset student assignments
  const resetAssignments = useCallback(() => {
    const updatedSeats = seats
      .filter((seat) => seat.assignedStudent != null)
      .map((seat) => ({ ...seat, assignedStudent: null }))
    mutation.mutate(updatedSeats)
  }, [seats, mutation])

  // Smart assign function
  const smartAssignStudents = useCallback(() => {
    const studentSeats = seats.filter((seat) => seat.assignedTutor && !seat.isTutorSeat).length
    if (studentSeats < developerWithProfiles.length) {
      setError(
        `Not enough student seats with tutors assigned. Need ${developerWithProfiles.length} seats, but only have ${studentSeats} (tutor seats excluded).`,
      )
      return
    }
    setError(null)
    const updatedSeats = smartAssign(seats, developerWithProfiles, peerAssignments)
    mutation.mutate(updatedSeats)
  }, [seats, developerWithProfiles, tutors, peerAssignments, mutation])

  // Download assignments as CSV
  const downloadAssignments = useDownloadAssignment(seats, developerWithProfiles, tutors)

  return (
    <Collapsible open={!isCollapsed} onOpenChange={() => setIsCollapsed(!isCollapsed)}>
      <Card>
        <CollapsibleTrigger asChild>
          <CardHeader className='cursor-pointer'>
            <div className='flex items-center justify-between'>
              <div>
                <CardTitle>Step 4: Student Assignment</CardTitle>
                <CardDescription>
                  Randomly assign students to seats with tutors, prioritizing Mac seats for students
                  without Macs
                </CardDescription>
              </div>
              <div className='flex items-center gap-2'>
                <Badge variant='secondary'>
                  <Users className='h-4 w-4 mr-1.5' />
                  {assignedStudents} of {totalStudents} Assigned
                </Badge>
                {isCollapsed ? (
                  <ChevronDown className='h-4 w-4' />
                ) : (
                  <ChevronUp className='h-4 w-4' />
                )}
              </div>
            </div>
          </CardHeader>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <CardContent className='space-y-4'>
            {error && (
              <Alert variant='destructive'>
                <AlertCircle className='h-4 w-4' />
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
            <div className='flex flex-col sm:flex-row gap-2 justify-between'>
              <div className='space-y-1'>
                <div className='text-sm font-medium'>Assignment Status</div>
                <div className='flex items-center'>
                  {assignmentStatus === 'none' && (
                    <Badge variant='secondary'>Not Assigned</Badge>
                  )}
                  {assignmentStatus === 'partial' && (
                    <Badge variant='outline'>
                      Partially Assigned ({assignedStudents}/{totalStudents})
                    </Badge>
                  )}
                  {assignmentStatus === 'complete' && (
                    <Badge variant='default'>
                      Fully Assigned ({assignedStudents}/{totalStudents})
                    </Badge>
                  )}
                </div>
              </div>
              <div className='flex flex-col sm:flex-row gap-2'>
                <Button
                  variant='outline'
                  onClick={downloadAssignments}
                  disabled={assignedStudents === 0}
                >
                  <Download className='mr-2 h-4 w-4' />
                  Download Assignments
                </Button>
                <ResetSeatAssignmentDialog
                  disabled={assignedStudents === 0}
                  onSuccess={resetAssignments}
                />
                <Button
                  onClick={assignStudents}
                  disabled={mutation.isPending || assignedStudents > 0}
                >
                  <UserCheck className='mr-2 h-4 w-4' />
                  Assign Random
                </Button>
                <Button
                  variant='secondary'
                  onClick={smartAssignStudents}
                  disabled={mutation.isPending || assignedStudents > 0}
                >
                  <Sparkles className='mr-2 h-4 w-4' />
                  Smart Assign
                </Button>
              </div>
            </div>
            {assignedStudents > 0 && (
              <Card className='overflow-hidden mt-4'>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Seat</TableHead>
                      <TableHead>Mac</TableHead>
                      <TableHead>Assigned Student</TableHead>
                      <TableHead>Student Has Mac</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {seats
                      .filter((seat) => seat.assignedStudent)
                      .sort((a, b) => a.seatName.localeCompare(b.seatName))
                      .map((seat) => {
                        const student = developerWithProfiles.find(
                          (dev) =>
                            dev.participation.courseParticipationID === seat.assignedStudent,
                        )
                        return (
                          <TableRow key={seat.seatName}>
                            <TableCell className='font-medium'>{seat.seatName}</TableCell>
                            <TableCell>
                              {seat.hasMac ? (
                                <Badge variant='secondary'>
                                  <Laptop className='h-3 w-3 mr-1' />
                                  Mac
                                </Badge>
                              ) : (
                                <span className='text-muted-foreground text-xs'>No</span>
                              )}
                            </TableCell>
                            <TableCell>
                              {student ? (
                                `${student.participation.student.firstName} ${student.participation.student.lastName}`
                              ) : (
                                <span className='text-muted-foreground'>Unknown</span>
                              )}
                            </TableCell>
                            <TableCell>
                              {student?.profile?.hasMacBook === true && (
                                <Badge variant='default'>
                                  <Laptop className='h-3 w-3 mr-1' />
                                  Yes
                                </Badge>
                              )}
                              {student?.profile?.hasMacBook === false && (
                                <Badge variant='destructive'>No</Badge>
                              )}
                              {student?.profile?.hasMacBook === undefined && (
                                <span className='text-muted-foreground text-xs'>Unknown</span>
                              )}
                            </TableCell>
                          </TableRow>
                        )
                      })}
                  </TableBody>
                </Table>
              </Card>
            )}
          </CardContent>
        </CollapsibleContent>
      </Card>
    </Collapsible>
  )
}
