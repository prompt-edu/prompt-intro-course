import { useState, useMemo, useCallback, useRef } from 'react'
import {
  ChevronDown,
  ChevronUp,
  Download,
  Upload,
  AlertCircle,
  UserCheck,
  Users,
  Laptop,
  Sparkles,
  X,
} from 'lucide-react'
import { useParams } from 'react-router-dom'
import { Seat } from '../../../../interfaces/Seat'
import { DeveloperWithProfile } from '../../interfaces/DeveloperWithProfile'
import { Tutor } from '../../../../interfaces/Tutor'
import { PeerAssignment } from '../../../../interfaces/PeerAssignment'
import { useUpdateSeats } from '../../hooks/useUpdateSeats'
import { useAssignStudents } from '../../hooks/useAssignStudents'
import { useDownloadAssignment } from '../../hooks/useDownloadAssignment'
import { smartAssign } from '../../utils/smartAssignment'
import { updatePeerAssignments } from '../../../../network/mutations/updatePeerAssignments'
import { updatePeerAssignments } from '../../../../network/mutations/updatePeerAssignments'
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
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

  // Smart assign function — fills remaining unassigned students into empty seats
  const smartAssignStudents = useCallback((resetStudentsFirst = false) => {
    let workingSeats = seats
    if (resetStudentsFirst) {
      workingSeats = seats.map((s) => ({ ...s, assignedStudent: null }))
    }

    // Auto-assign tutors if needed
    const hasTutors = workingSeats.some((s) => s.assignedTutor)
    if (!hasTutors && tutors.length > 0) {
      // Will be handled inside smartAssign
    }

    const currentAssigned = resetStudentsFirst ? 0 : assignedStudents
    const emptyStudentSeats = workingSeats.filter(
      (seat) => (seat.assignedTutor || !hasTutors) && !seat.isTutorSeat && !seat.assignedStudent,
    ).length
    const unassignedCount = developerWithProfiles.length - currentAssigned
    if (emptyStudentSeats < unassignedCount) {
      setError(
        `Not enough empty student seats. Need ${unassignedCount} seats, but only have ${emptyStudentSeats} available.`,
      )
      return
    }
    setError(null)
    const updatedSeats = smartAssign(workingSeats, developerWithProfiles, peerAssignments, tutors)
    mutation.mutate(updatedSeats)
  }, [seats, developerWithProfiles, assignedStudents, peerAssignments, tutors, mutation])

  // CSV import — matches by name, handles tutor seats and peer groups
  const { phaseId } = useParams<{ phaseId: string }>()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const handleImportCSV = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = async (e) => {
      const text = e.target?.result as string
      if (!text) return
      const lines = text.split('\n').filter((l) => l.trim())
      if (lines.length < 2) {
        setError('CSV file must have a header and at least one data row.')
        return
      }

      // Match exact column names from export
      const header = lines[0].split(',').map((h) => h.trim().toLowerCase())
      const findCol = (name: string) => header.findIndex((h) => h === name.toLowerCase())
      const seatCol = findCol('seat')
      const studentNameCol = findCol('assigned student')
      const tutorNameCol = findCol('assigned tutor')
      const tutorSeatCol = findCol('tutor seat')
      const peerGroupCol = findCol('peer group')

      if (seatCol < 0) {
        setError('CSV must have a "Seat" column.')
        return
      }

      // Build name -> ID lookups
      const studentNameToId = new Map<string, string>()
      for (const dev of developerWithProfiles) {
        const name = `${dev.participation.student.firstName} ${dev.participation.student.lastName}`
        studentNameToId.set(name.toLowerCase(), dev.participation.courseParticipationID)
      }
      const tutorNameToId = new Map<string, string>()
      if (tutors) {
        for (const t of tutors) {
          tutorNameToId.set(`${t.firstName} ${t.lastName}`.toLowerCase(), t.id)
        }
      }

      // Parse rows
      const updatedSeats = seats.map((s) => ({ ...s }))
      const seatIndex = new Map<string, number>()
      updatedSeats.forEach((s, i) => seatIndex.set(s.seatName, i))

      const peerGroups = new Map<string, string[]>()
      const warnings: string[] = []

      for (let i = 1; i < lines.length; i++) {
        const cols = lines[i].split(',').map((c) => c.trim().replace(/^"|"$/g, ''))
        const seatName = cols[seatCol]
        if (!seatName) continue
        const idx = seatIndex.get(seatName)
        if (idx === undefined) continue

        // Student
        const studentName = studentNameCol >= 0 ? (cols[studentNameCol] || '').trim() : ''
        let studentId: string | null = null
        if (studentName && studentName.toLowerCase() !== 'unassigned') {
          studentId = studentNameToId.get(studentName.toLowerCase()) ?? null
          if (!studentId) warnings.push(`Student "${studentName}" not found`)
        }
        updatedSeats[idx].assignedStudent = studentId

        // Tutor
        const tutorName = tutorNameCol >= 0 ? (cols[tutorNameCol] || '').trim() : ''
        if (tutorName && tutorName.toLowerCase() !== 'unknown tutor') {
          const tutorId = tutorNameToId.get(tutorName.toLowerCase()) ?? null
          if (tutorId) {
            updatedSeats[idx].assignedTutor = tutorId
          } else {
            warnings.push(`Tutor "${tutorName}" not found`)
          }
        }

        // Tutor seat flag
        if (tutorSeatCol >= 0) {
          updatedSeats[idx].isTutorSeat = (cols[tutorSeatCol] || '').toLowerCase() === 'yes'
        }

        // Collect peer groups
        if (peerGroupCol >= 0 && studentId) {
          const pg = (cols[peerGroupCol] || '').trim()
          if (pg) {
            if (!peerGroups.has(pg)) peerGroups.set(pg, [])
            peerGroups.get(pg)!.push(studentId)
          }
        }
      }

      // Save seat assignments
      mutation.mutate(updatedSeats)

      // Build and save peer assignments from groups
      if (peerGroups.size > 0 && phaseId) {
        const peerList: PeerAssignment[] = []
        for (const members of peerGroups.values()) {
          for (const a of members) {
            for (const b of members) {
              if (a !== b) peerList.push({ studentID: a, peerID: b })
            }
          }
        }
        try {
          await updatePeerAssignments(phaseId, peerList)
        } catch {
          warnings.push('Failed to save peer assignments')
        }
      }

      if (warnings.length > 0) {
        const unique = [...new Set(warnings)]
        setError(`Imported ${updatedSeats.length} seats. Warnings: ${unique.slice(0, 5).join('; ')}${unique.length > 5 ? ` (+${unique.length - 5} more)` : ''}`)
      } else {
        setError(null)
      }
    }
    reader.readAsText(file)
    if (fileInputRef.current) fileInputRef.current.value = ''
  }, [seats, developerWithProfiles, tutors, mutation, phaseId])

  // Download assignments as CSV
  const downloadAssignments = useDownloadAssignment(seats, developerWithProfiles, tutors, peerAssignments)

  // Unassigned students (for manual assignment)
  const unassignedStudents = useMemo(() => {
    const assignedSet = new Set(seats.filter((s) => s.assignedStudent).map((s) => s.assignedStudent!))
    return developerWithProfiles.filter(
      (dev) => !assignedSet.has(dev.participation.courseParticipationID),
    )
  }, [seats, developerWithProfiles])

  // Available seats (have tutor, not tutor seat, no student)
  const availableSeats = useMemo(() => {
    return seats
      .filter((s) => s.assignedTutor && !s.isTutorSeat && !s.assignedStudent)
      .sort((a, b) => a.seatName.localeCompare(b.seatName))
  }, [seats])

  // Manual assign a student to a seat
  const manualAssign = useCallback(
    (seatName: string, studentId: string) => {
      const seat = seats.find((s) => s.seatName === seatName)
      if (!seat) return
      mutation.mutate([{ ...seat, assignedStudent: studentId }])
    },
    [seats, mutation],
  )

  // Unassign a student from their seat
  const unassignStudent = useCallback(
    (seatName: string) => {
      const seat = seats.find((s) => s.seatName === seatName)
      if (!seat) return
      mutation.mutate([{ ...seat, assignedStudent: null }])
    },
    [seats, mutation],
  )

  return (
    <Collapsible open={!isCollapsed} onOpenChange={() => setIsCollapsed(!isCollapsed)}>
      <Card>
        <CollapsibleTrigger asChild>
          <CardHeader className='cursor-pointer'>
            <div className='flex items-center justify-between'>
              <div>
                <CardTitle>Step 4: Student Assignment</CardTitle>
                <CardDescription>
                  Assign students to seats automatically or manually. Mac seats are prioritized for
                  students without Macs.
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
              <div className='flex flex-wrap gap-2'>
                <input
                  type='file'
                  accept='.csv'
                  ref={fileInputRef}
                  className='hidden'
                  onChange={handleImportCSV}
                />
                <Button
                  variant='outline'
                  size='sm'
                  onClick={() => fileInputRef.current?.click()}
                >
                  <Upload className='mr-2 h-4 w-4' />
                  Import
                </Button>
                <Button
                  variant='outline'
                  size='sm'
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
                  size='sm'
                  onClick={assignStudents}
                  disabled={mutation.isPending || assignedStudents > 0}
                >
                  <UserCheck className='mr-2 h-4 w-4' />
                  Assign Random
                </Button>
                <Button
                  variant='secondary'
                  size='sm'
                  onClick={() => smartAssignStudents()}
                  disabled={mutation.isPending || assignedStudents >= totalStudents}
                >
                  <Sparkles className='mr-2 h-4 w-4' />
                  Smart Assign
                </Button>
                <Button
                  variant='secondary'
                  size='sm'
                  onClick={() => smartAssignStudents(true)}
                  disabled={mutation.isPending}
                >
                  <Sparkles className='mr-2 h-4 w-4' />
                  Reassign All
                </Button>
              </div>
            </div>

            {/* Manual assignment for unassigned students */}
            {unassignedStudents.length > 0 && availableSeats.length > 0 && (
              <Card className='overflow-hidden mt-4'>
                <CardHeader className='pb-2'>
                  <CardTitle className='text-sm'>Manual Assignment</CardTitle>
                  <CardDescription className='text-xs'>
                    Assign individual students to specific seats
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Student</TableHead>
                        <TableHead>Has Mac</TableHead>
                        <TableHead>Assign to Seat</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {unassignedStudents
                        .sort((a, b) =>
                          `${a.participation.student.lastName}`.localeCompare(
                            `${b.participation.student.lastName}`,
                          ),
                        )
                        .map((dev) => (
                          <TableRow key={dev.participation.courseParticipationID}>
                            <TableCell className='font-medium'>
                              {dev.participation.student.firstName}{' '}
                              {dev.participation.student.lastName}
                            </TableCell>
                            <TableCell>
                              {dev.profile?.hasMacBook === true && (
                                <Badge variant='default'>
                                  <Laptop className='h-3 w-3 mr-1' />
                                  Yes
                                </Badge>
                              )}
                              {dev.profile?.hasMacBook === false && (
                                <Badge variant='destructive'>No</Badge>
                              )}
                              {dev.profile?.hasMacBook === undefined && (
                                <span className='text-muted-foreground text-xs'>Unknown</span>
                              )}
                            </TableCell>
                            <TableCell>
                              <Select
                                onValueChange={(seatName) =>
                                  manualAssign(seatName, dev.participation.courseParticipationID)
                                }
                              >
                                <SelectTrigger className='w-40'>
                                  <SelectValue placeholder='Select seat' />
                                </SelectTrigger>
                                <SelectContent>
                                  {availableSeats.map((seat) => (
                                    <SelectItem key={seat.seatName} value={seat.seatName}>
                                      {seat.seatName}
                                      {seat.hasMac ? ' (Mac)' : ''}
                                    </SelectItem>
                                  ))}
                                </SelectContent>
                              </Select>
                            </TableCell>
                          </TableRow>
                        ))}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>
            )}

            {/* Current assignments table */}
            {assignedStudents > 0 && (
              <Card className='overflow-hidden mt-4'>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Seat</TableHead>
                      <TableHead>Mac</TableHead>
                      <TableHead>Assigned Student</TableHead>
                      <TableHead>Student Has Mac</TableHead>
                      <TableHead className='w-10'></TableHead>
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
                            <TableCell>
                              <Button
                                variant='ghost'
                                size='sm'
                                className='h-6 w-6 p-0 text-destructive hover:text-destructive'
                                onClick={() => unassignStudent(seat.seatName)}
                                disabled={mutation.isPending}
                                title='Remove student from seat'
                              >
                                <X className='h-3 w-3' />
                              </Button>
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
