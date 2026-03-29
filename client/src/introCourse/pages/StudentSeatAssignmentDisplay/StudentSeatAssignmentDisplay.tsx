import { useIntroCourseStore } from '../../zustand/useIntroCourseStore'
import { Monitor, User, Users, ExternalLink } from 'lucide-react'
import { getGravatarUrl } from '@/lib/getGravatarUrl'
import { Avatar, AvatarFallback, AvatarImage, Badge, Button } from '@tumaet/prompt-ui-components'
import { GITLAB_INTROCOURSE_BASE_URL } from '../../network/introCourseServerConfig'
import { useCourseStore } from '@tumaet/prompt-shared-state'
import { useParams } from 'react-router-dom'

export const StudentSeatAssignmentDisplay = () => {
  const { seatAssignment, peerAssignment } = useIntroCourseStore()
  const { courses } = useCourseStore()
  const { courseId } = useParams<{ courseId: string }>()
  const semesterTag = courses.find((c) => c.id === courseId)?.semesterTag ?? ''

  if (!seatAssignment) {
    return (
      <div className='text-center py-6 bg-muted/30 rounded-lg'>
        <User className='h-12 w-12 mx-auto text-muted-foreground mb-2' />
        <h3 className='text-lg font-medium mb-2'>No Seat Assigned</h3>
        <p className='text-muted-foreground max-w-md mx-auto'>
          You haven&apos;t been assigned a seat yet. Please check back later.
        </p>
      </div>
    )
  }

  const { seatName, hasMac, deviceID, tutorFirstName, tutorLastName, tutorEmail } = seatAssignment
  const tutorFullName = `${tutorFirstName} ${tutorLastName}`
  const tutorInitial = tutorFirstName.charAt(0)

  const hasPeers = peerAssignment && peerAssignment.peersIReview.length > 0

  return (
    <div className='grid grid-cols-1 md:grid-cols-2 gap-4'>
      {/* Seat Information Card */}
      <section className='p-4 bg-muted/20 rounded-lg shadow'>
        <div className='flex items-center gap-2 mb-3'>
          <Monitor className='h-5 w-5 text-primary' />
          <h2 className='text-lg font-medium'>Seat Information</h2>
        </div>
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
      </section>

      {/* Tutor Information Card */}
      <section className='p-4 bg-muted/20 rounded-lg shadow'>
        <div className='flex items-center gap-2 mb-3'>
          <User className='h-5 w-5 text-primary' />
          <h2 className='text-lg font-medium'>Your Tutor</h2>
        </div>
        <div className='flex items-center gap-4'>
          <Avatar className='h-16 w-16 border-2 border-background shadow-sm'>
            <AvatarImage src={getGravatarUrl(tutorEmail)} alt={`${tutorFullName}'s avatar`} />
            <AvatarFallback className='text-lg font-bold'>{tutorInitial}</AvatarFallback>
          </Avatar>
          <div>
            <p className='font-semibold text-lg'>{tutorFullName}</p>
            <p className='text-sm text-muted-foreground'>{tutorEmail}</p>
          </div>
        </div>
      </section>

      {/* Peer Review Information Card */}
      {hasPeers && (
        <section className='p-4 bg-muted/20 rounded-lg shadow md:col-span-2'>
          <div className='flex items-center gap-2 mb-3'>
            <Users className='h-5 w-5 text-primary' />
            <h2 className='text-lg font-medium'>Your Review Peers</h2>
          </div>
          <p className='text-sm text-muted-foreground mb-4'>
            You review each other&apos;s merge requests. Follow the testing steps in the MR
            description and verify functionality.
          </p>
          <div className='grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3'>
            {peerAssignment.peersIReview.map((peer) => (
              <div
                key={peer.courseParticipationID}
                className='border rounded-md p-3 flex flex-col gap-2'
              >
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
              </div>
            ))}
          </div>
        </section>
      )}
    </div>
  )
}
