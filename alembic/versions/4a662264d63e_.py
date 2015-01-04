"""empty message

Revision ID: 4a662264d63e
Revises: 42f6e7d2c9af
Create Date: 2015-01-04 16:31:59.802686

"""

# revision identifiers, used by Alembic.
revision = '4a662264d63e'
down_revision = '42f6e7d2c9af'

from alembic import op
import sqlalchemy as sa


def upgrade():
    ### commands auto generated by Alembic - please adjust! ###
    op.add_column('recordings', sa.Column('watchable', sa.Boolean(), nullable=True))
    ### end Alembic commands ###


def downgrade():
    ### commands auto generated by Alembic - please adjust! ###
    op.drop_column('recordings', 'watchable')
    ### end Alembic commands ###
