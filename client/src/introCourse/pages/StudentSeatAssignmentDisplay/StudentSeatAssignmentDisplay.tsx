import { getGravatarUrl, useCourseStore } from '@tumaet/prompt-shared-state'
import {
  Avatar,
  AvatarFallback,
  AvatarImage,
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@tumaet/prompt-ui-components'
import { ExternalLink, Monitor, User, Users } from 'lucide-react'
import { useParams } from 'react-router-dom'
import { GITLAB_INTROCOURSE_BASE_URL } from '../../network/introCourseServerConfig'
import { useIntroCourseStore } from '../../zustand/useIntroCourseStore'

export const StudentSeatAssignmentDisplay = () => {
  const { seatAssignment, peerAssignment } = useIntroCourseStore()
  const { courses } = useCourseStore()
  const { courseId } = useParams<{ courseId: string }>()
  const semesterTag = courses.find((c) => c.id === courseId)?.semesterTag ?? ''

  if (!seatAssignment) {
    return (
      <Card>
        <CardContent className='text-center py-8'>
          <User className='h-12 w-12 mx-auto text-muted-foreground mb-2' />
          <h3 className='text-lg font-medium mb-2'>No Seat Assigned</h3>
          <p className='text-muted-foreground max-w-md mx-auto'>
            You haven&apos;t been assigned a seat yet. Please check back later.
          </p>
        </CardContent>
      </Card>
    )
  }

  const { seatName, hasMac, deviceID, tutorFirstName, tutorLastName, tutorEmail } = seatAssignment
  const tutorFullName = `${tutorFirstName} ${tutorLastName}`
  const tutorInitial = tutorFirstName.charAt(0)

  const hasPeers = peerAssignment && peerAssignment.peersIReview.length > 0

  return (
    <div className='grid grid-cols-1 md:grid-cols-2 gap-4'>
      {/* Seat Information Card */}
      <Card>
        <CardHeader className='pb-3'>
          <CardTitle className='flex items-center gap-2 text-base'>
            <Monitor className='h-5 w-5 text-primary' />
            Seat Information
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div>
            <p className='text-sm text-muted-foreground'>Assigned Seat</p>
            <p className='text-xl font-semibold'>{seatName}</p>
          </div>
          <div className='mt-4'>
            <p className='text-sm text-muted-foreground'>Device Type</p>
            <div className='flex items-center gap-2 mt-1'>
              <Badge variant='outline'>{hasMac ? 'Chair Mac' : 'Own MacBook'}</Badge>
              {deviceID && <span className='text-sm'>ID: {deviceID}</span>}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Tutor Information Card */}
      <Card>
        <CardHeader className='pb-3'>
          <CardTitle className='flex items-center gap-2 text-base'>
            <User className='h-5 w-5 text-primary' />
            Your Tutor
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className='flex items-center gap-4'>
            <Avatar className='h-16 w-16 border-2 border-background shadow-xs'>
              <AvatarImage src={getGravatarUrl(tutorEmail)} alt={`${tutorFullName}'s avatar`} />
              <AvatarFallback className='text-lg font-bold'>{tutorInitial}</AvatarFallback>
            </Avatar>
            <div>
              <p className='font-semibold text-lg'>{tutorFullName}</p>
              <p className='text-sm text-muted-foreground'>{tutorEmail}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Peer Review Information Card */}
      {hasPeers && (
        <Card className='md:col-span-2'>
          <CardHeader className='pb-3'>
            <CardTitle className='flex items-center gap-2 text-base'>
              <Users className='h-5 w-5 text-primary' />
              Your Review Peers
            </CardTitle>
            <CardDescription>
              Your main task is to <strong>manually test</strong> your peer&apos;s application:
              follow the testing steps in the MR description and verify that the functionality works
              as described. Code reviewing is optional but encouraged.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className='grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3'>
              {peerAssignment.peersIReview.map((peer) => (
                <Card key={peer.courseParticipationID}>
                  <CardContent className='p-3 flex flex-col gap-2'>
                    <p className='font-medium'>{peer.gitlabUsername}</p>
                    {peer.seatName && (
                      <p className='text-sm text-muted-foreground'>Seat: {peer.seatName}</p>
                    )}
                    {peer.gitlabUsername && peer.tutorGitlabUsername && (
                      <Button
                        variant='outline'
                        size='sm'
                        className='w-fit'
                        onClick={() =>
                          window.open(
                            `${GITLAB_INTROCOURSE_BASE_URL}/ase/iPraktikum/${semesterTag}/Introcourse/${peer.tutorGitlabUsername}/${peer.gitlabUsername}`,
                            '_blank',
                          )
                        }
                      >
                        <ExternalLink className='h-3 w-3 mr-1' />
                        GitLab Repo
                      </Button>
                    )}
                  </CardContent>
                </Card>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
